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

type OpenAIProvider struct {
	client *openai.Client
	agent  *Agent // Back-reference to access config and tools
}

func NewOpenAIProvider(client *openai.Client, agent *Agent) *OpenAIProvider {
	return &OpenAIProvider{
		client: client,
		agent:  agent,
	}
}

func (p *OpenAIProvider) newOpenAIMessage() openai.ChatCompletionRequest {
	a := p.agent
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

func (p *OpenAIProvider) Completion(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error) {
	// Prepare messages for OpenAI
	body := p.newOpenAIMessage()
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
	if p.client == nil {
		return result, stopReason, fmt.Errorf("openAI client is not initialized")
	}
	stream, err := p.client.CreateChatCompletionStream(ctx, body)
	if err != nil {
		return result, stopReason, err
	}

	defer stream.Close()

	contentItem := openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleAssistant,
	}

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
				contentItem.Content += delta.Content
				if len(contentItem.MultiContent) > 0 && contentItem.MultiContent[len(contentItem.MultiContent)-1].Type == openai.ChatMessagePartTypeText {
					contentItem.MultiContent[len(contentItem.MultiContent)-1].Text += delta.Content
				} else {
					contentItem.MultiContent = append(contentItem.MultiContent, openai.ChatMessagePart{Type: openai.ChatMessagePartTypeText, Text: delta.Content})
				}
				if callback != nil {
					callback(ChatEvent{Type: EventTypeText, Content: delta.Content, IsEnd: false})
				}
			case len(delta.ToolCalls) > 0:
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

func (p *OpenAIProvider) ToolUse(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) ([]database.MessageUnion, error) {
	a := p.agent
	lastMessage := message.OfOpenAI
	if lastMessage == nil {
		return nil, fmt.Errorf("last message is not an OpenAI message")
	}
	fmt.Printf("toolUseOpenAI: processing tool calls (count: %d)\n", len(lastMessage.ToolCalls))

	// Find the tool use block
	toolUseBlocks := lastMessage.ToolCalls
	if len(toolUseBlocks) == 0 {
		fmt.Println("No tool use blocks found in chat history", "sessionId", a.session.ID, "agentName", a.name)
		return nil, fmt.Errorf("no tool use blocks found in chat history")
	}

	results := make([]database.MessageUnion, 0, len(toolUseBlocks))

	for _, toolUse := range toolUseBlocks {
		fmt.Printf("toolUseOpenAI: invoking tool (name: %s, id: %s)\n", toolUse.Function.Name, toolUse.ID)

		// Callback: Start tool execution
		if callback != nil {
			callback(ChatEvent{
				Type:    EventTypeToolUse,
				ToolUse: ToolCallWrapper{OpenAI: toolUse},
			})
		}

		// Normally toolUse.Name will have format <mcp>/<tool_name>
		parts := strings.SplitN(toolUse.Function.Name, "--", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid tool name format, expected <mcp>--<tool_name>", "sessionId", a.session.ID, "agentName", a.name, "tool_name", toolUse.Function.Name)
			return nil, fmt.Errorf("invalid tool name format, expected <mcp>--<tool_name>")
		}
		mcpName := parts[0]
		toolName := parts[1]
		mcpClient, ok := a.mcpClients[mcpName]
		if !ok {
			fmt.Println("MCP client not found", "sessionId", a.session.ID, "agentName", a.name, "mcpName", mcpName)
			return nil, fmt.Errorf("MCP client not found: %s", mcpName)
		}

		// Parse arguments
		var arguments map[string]interface{}
		if toolUse.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(toolUse.Function.Arguments), &arguments); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
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
			return nil, fmt.Errorf("failed to call tool %s: %w", toolUse.Function.Name, err)
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

		// OpenAI tool messages require Content as a plain string, not MultiContent
		toolResult := openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			ToolCallID: toolUse.ID,
			Content:    contentBuilder.String(),
			Name:       toolUse.Function.Name,
		}

		// Callback: End tool execution
		if callback != nil {
			callback(ChatEvent{
				Type:       EventTypeToolResult,
				ToolUse:    ToolCallWrapper{OpenAI: toolUse},
				ToolResult: ToolResultWrapper{OpenAI: toolResult},
			})
		}

		results = append(results, database.MessageUnion{OfOpenAI: &toolResult})
		fmt.Printf("toolUseOpenAI: tool executed successfully\n")
	}
	return results, nil
}
