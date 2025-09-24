package agent

import (
	"context"
	"encoding/json"
	"fmt"
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
	}

	var tools []openai.Tool
	for _, tool := range a.config.Tools {
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

func (a *Agent) completionOpenAI(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error) {
	// Prepare messages for OpenAI
	body := a.newOpenAIMessage()
	for _, m := range messages {
		if am := m.ToAnthropic(); am != nil {
			body.Messages = append(body.Messages, *am)
		}
	}
	result := database.MessageUnion{
		OfOpenAI: &openai.ChatCompletionMessage{},
	}
	var stopReason database.StopReason
	// Call OpenAI API
	if a.provider == nil || a.provider.OfOpenAI == nil {
		return result, stopReason, fmt.Errorf("openAI client is not initialized")
	}
	stream := a.provider.OfOpenAI.Messages.NewStreaming(ctx, body)
	if err := stream.Err(); err != nil {
		return result, stopReason, err
	}
	toolUseInput := strings.Builder{}
	contentBlockType := ""
	contentItem := openai.ContentBlockParamUnion{}
	// Handle the stream
	for stream.Next() {
		chunk := stream.Current()
		// Handle the chunk
		switch chunk.Type {
		case "message_start":
			msg := chunk.AsMessageStart()
			fmt.Println("Message Start",
				"sessionId", a.session.ID,
				"agentName", a.name,
				"message_id", msg.Message.ID,
				"inputToken", msg.Message.Usage.InputTokens,
				"cachedCreationInputToken", msg.Message.Usage.CacheCreationInputTokens,
				"cachedReadInputToken", msg.Message.Usage.CacheReadInputTokens)
		case "message_delta":
			msg := chunk.AsMessageDelta()
			fmt.Println("Message Delta", "sessionId", a.session.ID, "agentName", a.name, "stop_reason", msg.Delta.StopReason, "outputTokens", msg.Usage.OutputTokens, "stop_sequence", msg.Delta.StopSequence)
			stopReason = anthropicToDbStopReason(msg.Delta.StopReason)
		case "message_stop":
			_ = chunk.AsMessageStop()
			fmt.Println("Message Stop", "sessionId", a.session.ID, "agentName", a.name)
		case "content_block_start":
			msg := chunk.AsContentBlockStart()
			fmt.Println("Content Block Start", "sessionId", a.session.ID, "agentName", a.name, "block_type", msg.ContentBlock.Type)
			contentBlockType = msg.ContentBlock.Type
			switch msg.ContentBlock.Type {
			case "text":
				text := msg.ContentBlock.AsText()
				contentItem.OfText = &openai.TextBlockParam{Text: text.Text}
			// case "thinking":
			// 	thinking := msg.ContentBlock.AsThinking()
			// 	// If callback is provided, call it with initial thinking content
			// 	if callback != nil {
			// 		_ = callback(thinking.Thinking, true, false)
			// 	}
			// 	contentItem.OfThinking = &openai.ThinkingBlockParam{
			// 		Thinking: thinking.Thinking,
			// 	}
			case "tool_use":
				toolUse := msg.ContentBlock.AsToolUse()
				fmt.Println("Tool Use", "sessionId", a.session.ID, "agentName", a.name, "id", toolUse.ID, "tool", toolUse.Name, "input", string(toolUse.Input))
				toolUseInput.Reset()
				// Initialize tool use block
				contentItem.OfToolUse = &openai.ToolUseBlockParam{
					ID:           msg.ContentBlock.ID,
					Name:         msg.ContentBlock.Name,
					CacheControl: toolUse.ToParam().CacheControl,
					Input:        map[string]any{},
				}
			}
		case "content_block_delta":
			msg := chunk.AsContentBlockDelta()
			switch msg.Delta.Type {
			case "thinking_delta":
				delta := msg.Delta.AsThinkingDelta()
				if callback != nil {
					_ = callback(delta.Thinking, true, false)
				}
				contentItem.OfThinking.Thinking += delta.Thinking
			case "signature_delta":
				delta := msg.Delta.AsSignatureDelta()
				if contentItem.OfThinking != nil {
					contentItem.OfThinking.Signature += delta.Signature
				}
			case "text_delta":
				delta := msg.Delta.AsTextDelta()
				if callback != nil {
					_ = callback(delta.Text, false, false)
				}
				contentItem.OfText.Text += delta.Text
			case "input_json_delta":
				delta := msg.Delta.AsInputJSONDelta()
				toolUseInput.WriteString(delta.PartialJSON)
			}
		case "content_block_stop":
			_ = chunk.AsContentBlockStop()
			msg := chunk.AsContentBlockStop()
			if contentBlockType == "tool_use" {
				tool_use_input := map[string]any{}
				err := json.Unmarshal([]byte(toolUseInput.String()), &tool_use_input)
				if err != nil {
					return result, stopReason, fmt.Errorf("failed to unmarshal tool use input: %w", err)
				}
				contentItem.OfToolUse.Input = tool_use_input
				toolUseInput.Reset()
			}
			_ = callback("", false, true)
			result.OfOpenAI.Content = append(result.OfOpenAI.Content, contentItem)
			contentItem = openai.ContentBlockParamUnion{}
			fmt.Println("Content Block Stop", "sessionId", a.session.ID, "agentName", a.name, "index", msg.Index)
		}
	}

	return result, stopReason, nil
}

func (a *Agent) toolUseOpenAI(ctx context.Context, message *database.MessageUnion) (database.MessageUnion, error) {
	lastMessage := message.ToAnthropic()
	result := database.MessageUnion{}
	if lastMessage == nil {
		return result, fmt.Errorf("last message is not an Anthropic message")
	}
	// Find the tool use block
	toolUseBlocks := []openai.ToolUseBlockParam{}

	for _, block := range lastMessage.Content {
		if block.OfToolUse != nil {
			toolUseBlocks = append(toolUseBlocks, *block.OfToolUse)
		}
	}
	if len(toolUseBlocks) == 0 {
		fmt.Println("No tool use blocks found in chat history", "sessionId", a.session.ID, "agentName", a.name)
		return result, fmt.Errorf("no tool use blocks found in chat history")
	}
	toolUseMessage := openai.MessageParam{
		Role:    openai.MessageParamRoleUser,
		Content: []openai.ContentBlockParamUnion{},
	}
	for _, toolUse := range toolUseBlocks {
		fmt.Println("Invoking tool", "name", toolUse.Name, "input", toolUse.Input)
		// Normally toolUse.Name will have format <mcp>/<tool_name>
		parts := strings.SplitN(toolUse.Name, "--", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid tool name format, expected <mcp>--<tool_name>", "sessionId", a.session.ID, "agentName", a.name, "tool_name", toolUse.Name)
			return result, fmt.Errorf("invalid tool name format, expected <mcp>--<tool_name>")
		}
		mcpName := parts[0]
		toolName := parts[1]
		mcpClient, ok := a.mcpClients[mcpName]
		if !ok {
			fmt.Println("MCP client not found", "sessionId", a.session.ID, "agentName", a.name, "mcpName", mcpName)
			return result, fmt.Errorf("MCP client not found: %s", mcpName)
		}
		// Serialize the input JSON into map[string] any
		toolResponse, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: toolUse.Input,
				Meta: &mcp.Meta{
					AdditionalFields: map[string]any{
						"user_id":    a.session.UserID.String(),
						"session_id": a.session.ID.String(),
					},
				},
			},
			// Header: http.Header{
			// 	"X-Session-ID": []string{a.session.ID.String()},
			// 	"X-User-ID":    []string{a.session.UserID.String()},
			// },
		})
		if err != nil {
			fmt.Println("Failed to call tool", "sessionId", a.session.ID, "agentName", a.name, "toolName", toolUse.Name, "error", err)
			return result, fmt.Errorf("failed to call tool %s: %w", toolUse.Name, err)
		}

		toolResult := openai.ContentBlockParamUnion{
			OfToolResult: &openai.ToolResultBlockParam{
				ToolUseID: toolUse.ID,
				Content:   []openai.ToolResultBlockParamContentUnion{},
				IsError:   openai.Bool(toolResponse.IsError),
			},
		}
		// Convert the tool response content to anthropic format
		for _, content := range toolResponse.Content {
			switch content := content.(type) {
			case mcp.TextContent:
				toolResult.OfToolResult.Content = append(
					toolResult.OfToolResult.Content,
					openai.ToolResultBlockParamContentUnion{OfText: &openai.TextBlockParam{Text: content.Text}},
				)
				fmt.Println("Tool result: ", "sessionId", a.session.ID, "agentName", a.name, "tool_id", toolUse.ID, "tool_name", toolUse.Name, "text", content.Text)
			}
		}
		toolUseMessage.Content = append(toolUseMessage.Content, toolResult)
	}
	result.OfOpenAI = &toolUseMessage
	return result, nil
}
