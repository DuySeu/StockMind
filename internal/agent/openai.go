package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"stockmind/internal/database"

	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"
)

func createOpenAIClient(config OpenAIConfig) (*LLMClientWrapper, error) {
	var openaiClient *openai.Client

	if config.AuthType == "openai" {
		openaiClient = openai.NewClient(config.APIKey)
	}
	if config.AuthType == "open_router" {
		var defaultConfig openai.ClientConfig
		key := config.APIKey
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is not found")
		}
		defaultConfig = openai.DefaultConfig(key)
		defaultConfig.BaseURL = config.BaseURL

		openaiClient = openai.NewClientWithConfig(defaultConfig)
	}
	return &LLMClientWrapper{OfOpenAI: openaiClient}, nil
}

func (a *Agent) newOpenAIMessage() openai.ChatCompletionRequest {
	request := openai.ChatCompletionRequest{
		Model:       a.config.ModelID,
		MaxTokens:   int(a.config.MaxTokens),
		Temperature: float32(a.config.Temperature),
		Stream:      true,
	}

	var tools []openai.Tool
	for _, tool := range a.tools {
		openAITool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		}
		tools = append(tools, openAITool)
	}
	request.Tools = tools
	request.Messages = []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: a.config.SystemPrompt},
	}
	return request
}
func openaiToDbStopReason(reason openai.FinishReason) database.StopReason {
	switch reason {
	case openai.FinishReasonLength: // Max tokens
		return database.StopReasonMaxTokens
	case openai.FinishReasonToolCalls: // Tool call
		return database.StopReasonToolCall
	case openai.FinishReasonStop: // End Turn
		return database.StopReasonAgentDone
	default:
		return database.StopReasonUnknown
	}
}

// processThinkingContent processes streaming content to detect and separate
// thinking blocks (wrapped in <think>...</think> tags) from regular content.
// It handles partial tags across stream chunks using a buffer.
// Returns the content to emit and whether it's thinking content.
func processThinkingContent(chunk string, buffer *strings.Builder, inThinking *bool) (string, bool) {
	// Add chunk to buffer for processing
	buffer.WriteString(chunk)
	content := buffer.String()

	var result strings.Builder
	isThinking := *inThinking

	for len(content) > 0 {
		if *inThinking {
			// Look for end of thinking block
			endIdx := strings.Index(content, "</think>")
			if endIdx != -1 {
				// Found end tag - emit thinking content up to it
				result.WriteString(content[:endIdx])
				content = content[endIdx+8:] // Skip </think>
				*inThinking = false
				isThinking = true // This chunk contained thinking content
			} else {
				// Check if we might have a partial end tag at the end
				if len(content) < 8 && strings.HasPrefix("</think>", content) {
					// Keep in buffer for next chunk
					buffer.Reset()
					buffer.WriteString(content)
					break
				}
				// No end tag found, emit all as thinking content
				result.WriteString(content)
				buffer.Reset()
				return result.String(), true
			}
		} else {
			// Look for start of thinking block
			startIdx := strings.Index(content, "<think>")
			if startIdx != -1 {
				// Found start tag - emit regular content up to it
				if startIdx > 0 {
					result.WriteString(content[:startIdx])
					content = content[startIdx:]
					buffer.Reset()
					buffer.WriteString(content)
					if result.Len() > 0 {
						return result.String(), false
					}
				}
				content = content[7:] // Skip <think>
				*inThinking = true
			} else {
				// Check if we might have a partial start tag at the end
				for i := 1; i < 7 && i < len(content); i++ {
					suffix := content[len(content)-i:]
					if strings.HasPrefix("<think>", suffix) {
						// Keep partial tag in buffer for next chunk
						result.WriteString(content[:len(content)-i])
						buffer.Reset()
						buffer.WriteString(suffix)
						if result.Len() > 0 {
							return result.String(), false
						}
						return "", false
					}
				}
				// No start tag found, emit all as regular content
				result.WriteString(content)
				buffer.Reset()
				return result.String(), false
			}
		}
	}

	buffer.Reset()
	return result.String(), isThinking
}

func (a *Agent) completionOpenAI(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error) {
	// Prepare messages for OpenAI
	body := a.newOpenAIMessage()
	for _, m := range messages {
		if am := m.OfOpenAI; am != nil {
			body.Messages = append(body.Messages, *am)
		}
	}
	result := database.MessageUnion{
		OfOpenAI: &openai.ChatCompletionMessage{
			Role:         openai.ChatMessageRoleAssistant,
			MultiContent: []openai.ChatMessagePart{},
			ToolCalls:    []openai.ToolCall{},
		},
	}
	var stopReason database.StopReason
	// Call OpenAI API
	if a.provider == nil || a.provider.OfOpenAI == nil {
		return result, stopReason, fmt.Errorf("openAI client is not initialized")
	}
	stream, err := a.provider.OfOpenAI.CreateChatCompletionStream(ctx, body)
	if err != nil {
		return result, stopReason, err
	}

	defer stream.Close()

	contentItem := openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleAssistant,
	}

	// Thinking state tracker for models that use <think>...</think> tags
	// (e.g., DeepSeek R1, Qwen3 with thinking mode)
	inThinkingBlock := false
	var contentBuffer strings.Builder

	// Handle the stream
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// fmt.Println("\nStream finished")
			// Call callback to signal end of block
			if callback != nil {
				callback(ChatEvent{IsEnd: true})
			}
			result.OfOpenAI = &contentItem
			return result, stopReason, nil
		}

		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return result, stopReason, err
		}

		for _, chunk := range response.Choices {
			// Handle the chunk
			delta := chunk.Delta
			if chunk.FinishReason != "" {
				stopReason = openaiToDbStopReason(chunk.FinishReason)
			}
			switch {
			case delta.Content != "":
				// Process content for thinking tags
				processedContent, isThinking := processThinkingContent(delta.Content, &contentBuffer, &inThinkingBlock)

				// Accumulate content in the message
				if len(contentItem.MultiContent) > 0 && contentItem.MultiContent[len(contentItem.MultiContent)-1].Type == openai.ChatMessagePartTypeText {
					contentItem.MultiContent[len(contentItem.MultiContent)-1].Text += delta.Content
				} else {
					contentItem.MultiContent = append(contentItem.MultiContent, openai.ChatMessagePart{Type: openai.ChatMessagePartTypeText, Text: delta.Content})
				}

				// Send to callback with thinking flag
				if callback != nil && processedContent != "" {
					eventType := EventTypeText
					if isThinking {
						eventType = EventTypeThinking
					}
					callback(ChatEvent{
						Type:    eventType,
						Content: processedContent,
					})
				}
			case len(delta.ToolCalls) > 0:
				// Append tool calls
				// Note: OpenAI stream returns partial tool calls, we need to accumulate them
				// But for now, let's assume we just collect them
				// Complex tool call accumulation logic might be needed if using official go-openai stream support for tools
				// For simplicity, we might need to rely on the final accumulated result if the library handles it,
				// but standard stream handling requires manual accumulation.
				// Let's implement basic accumulation if needed, or just rely on what we have.
				// Actually, go-openai's Delta.ToolCalls usually comes with Index.
				// We need to accumulate them properly.

				for _, tc := range delta.ToolCalls {
					if tc.Index != nil {
						index := *tc.Index
						if index >= len(contentItem.ToolCalls) {
							// Expand slice
							for i := len(contentItem.ToolCalls); i <= index; i++ {
								contentItem.ToolCalls = append(contentItem.ToolCalls, openai.ToolCall{})
							}
						}
						if tc.ID != "" {
							contentItem.ToolCalls[index].ID = tc.ID
							contentItem.ToolCalls[index].Type = tc.Type
						}
						if tc.Function.Name != "" {
							if contentItem.ToolCalls[index].Function.Name == "" {
								contentItem.ToolCalls[index].Function.Name = tc.Function.Name
							} else {
								contentItem.ToolCalls[index].Function.Name += tc.Function.Name
							}
						}
						if tc.Function.Arguments != "" {
							contentItem.ToolCalls[index].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			default:
				continue
			}
		}
	}
}

func (a *Agent) toolUseOpenAI(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) (database.MessageUnion, error) {
	lastMessage := message.OfOpenAI
	result := database.MessageUnion{}
	if lastMessage == nil {
		return result, fmt.Errorf("last message is not an OpenAI message")
	}
	fmt.Printf("toolUseOpenAI: processing tool calls (count: %d)\n", len(lastMessage.ToolCalls))

	// Find the tool use block
	toolUseBlocks := lastMessage.ToolCalls
	if len(toolUseBlocks) == 0 {
		fmt.Println("No tool use blocks found in chat history", "sessionId", a.session.ID, "agentName", a.name)
		return result, fmt.Errorf("no tool use blocks found in chat history")
	}

	// For now, we only support one tool call per turn because MessageUnion only supports one message
	// TODO: Support multiple tool calls by updating MessageUnion to support list of messages
	if len(toolUseBlocks) > 1 {
		fmt.Println("Warning: Multiple tool calls detected, but only the first one will be processed correctly due to current limitation")
	}

	// We will process the first tool call and return it as the result
	toolUse := toolUseBlocks[0]
	fmt.Printf("toolUseOpenAI: invoking tool (name: %s, id: %s)\n", toolUse.Function.Name, toolUse.ID)

	// Callback: Start tool execution
	if callback != nil {
		callback(ChatEvent{
			Type:    EventTypeToolUse,
			ToolUse: toolUse,
		})
	}

	// Normally toolUse.Name will have format <mcp>/<tool_name>
	parts := strings.SplitN(toolUse.Function.Name, "--", 2)
	if len(parts) != 2 {
		fmt.Println("Invalid tool name format, expected <mcp>--<tool_name>", "sessionId", a.session.ID, "agentName", a.name, "tool_name", toolUse.Function.Name)
		return result, fmt.Errorf("invalid tool name format, expected <mcp>--<tool_name>")
	}
	mcpName := parts[0]
	toolName := parts[1]
	mcpClient, ok := a.mcpClients[mcpName]
	if !ok {
		fmt.Println("MCP client not found", "sessionId", a.session.ID, "agentName", a.name, "mcpName", mcpName)
		return result, fmt.Errorf("MCP client not found: %s", mcpName)
	}

	// Parse arguments
	var arguments map[string]interface{}
	if toolUse.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolUse.Function.Arguments), &arguments); err != nil {
			return result, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
		}
	} else {
		arguments = make(map[string]interface{})
	}

	// Serialize the input JSON into map[string] any
	toolResponse, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
			Meta: &mcp.Meta{
				AdditionalFields: map[string]any{
					"user_id":    a.session.CreatedBy,
					"session_id": a.session.ID,
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("toolUseOpenAI: tool invocation failed (tool: %s, error: %v)\n",
			toolUse.Function.Name, err)
		return result, fmt.Errorf("failed to call tool %s: %w", toolUse.Function.Name, err)
	}

	toolResult := openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleTool,
		ToolCallID:   toolUse.ID,
		MultiContent: []openai.ChatMessagePart{},
		Name:         toolUse.Function.Name,
	}

	// Convert the tool response content to string
	var contentBuilder strings.Builder
	for _, content := range toolResponse.Content {
		switch content := content.(type) {
		case mcp.TextContent:
			contentBuilder.WriteString(content.Text)
			fmt.Println("Tool result: ", "sessionId", a.session.ID, "agentName", a.name, "tool_id", toolUse.ID, "tool_name", toolUse.Function.Name, "text", content.Text)
		}
	}
	toolResult.MultiContent = []openai.ChatMessagePart{
		{Type: openai.ChatMessagePartTypeText, Text: contentBuilder.String()},
	}

	// Callback: End tool execution
	if callback != nil {
		callback(ChatEvent{
			Type:       EventTypeToolResult,
			ToolUse:    toolUse,
			ToolResult: toolResult,
		})
	}

	result.OfOpenAI = &toolResult
	fmt.Printf("toolUseOpenAI: tool executed successfully\n")
	return result, nil
}
