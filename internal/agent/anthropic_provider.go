package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"stockmind/internal/database"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/mark3labs/mcp-go/mcp"
)

type AnthropicProvider struct {
	client *anthropic.Client
	agent  *Agent
}

func NewAnthropicProvider(client *anthropic.Client) *AnthropicProvider {
	return &AnthropicProvider{
		client: client,
	}
}

func (p *AnthropicProvider) SetAgent(agent *Agent) {
	p.agent = agent
}
func (p *AnthropicProvider) newAnthropicMessage() anthropic.MessageNewParams {
	a := p.agent
	var msgParams anthropic.MessageNewParams
	if a.config.ThinkingToken == 0 {
		disabledThinking := anthropic.NewThinkingConfigDisabledParam()
		msgParams = anthropic.MessageNewParams{
			Model:       anthropic.Model(a.config.ModelID),
			MaxTokens:   a.config.MaxTokens,
			Thinking:    anthropic.ThinkingConfigParamUnion{OfDisabled: &disabledThinking},
			Temperature: anthropic.Float(a.config.Temperature),
			TopP:        anthropic.Float(a.config.TopP),
			TopK:        anthropic.Int(a.config.TopK),
		}
	} else {
		msgParams = anthropic.MessageNewParams{
			Model:       anthropic.Model(a.config.ModelID),
			MaxTokens:   a.config.MaxTokens,
			Thinking:    anthropic.ThinkingConfigParamOfEnabled(a.config.ThinkingToken),
			Temperature: anthropic.Float(1.0),
		}
	}
	// Transform tools to ToolUnionParam
	var tools []anthropic.ToolUnionParam
	for _, tool := range a.tools {
		anthropicTool := anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type:       "object",
				Properties: tool.InputSchema.Properties,
				Required:   tool.InputSchema.Required,
			},
			Type: "custom",
		}
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &anthropicTool})
	}
	msgParams.Tools = tools
	msgParams.System = []anthropic.TextBlockParam{
		{Text: a.config.SystemPrompt},
	}
	return msgParams
}

func anthropicToDbStopReason(sr anthropic.StopReason) database.StopReason {
	switch sr {
	case anthropic.StopReasonMaxTokens:
		return database.StopReasonMaxTokens
	case anthropic.StopReasonToolUse:
		return database.StopReasonToolCall
	case anthropic.StopReasonEndTurn:
		return database.StopReasonAgentDone
	default:
		// Not handle for now
		return database.StopReasonUnknown
	}
}

// Assume Streaming by Default
// Complete the turn with streaming
func (p *AnthropicProvider) Completion(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error) {
	a := p.agent
	// Prepare messages for Anthropics
	body := p.newAnthropicMessage()
	for _, m := range messages {
		if am := m.OfAnthropic; am != nil {
			body.Messages = append(body.Messages, *am)
		}
	}
	result := database.MessageUnion{
		OfAnthropic: &anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: []anthropic.ContentBlockParamUnion{},
		},
	}
	var stopReason database.StopReason
	// Call Anthropics API
	if p.client == nil {
		return result, stopReason, fmt.Errorf("Anthropic client is not initialized")
	}
	stream := p.client.Messages.NewStreaming(ctx, body)
	if err := stream.Err(); err != nil {
		return result, stopReason, err
	}
	toolUseInput := strings.Builder{}
	contentBlockType := ""
	contentItem := anthropic.ContentBlockParamUnion{}
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
				contentItem.OfText = &anthropic.TextBlockParam{Text: text.Text}
			case "thinking":
				thinking := msg.ContentBlock.AsThinking()
				// If callback is provided, call it with initial thinking content
				if callback != nil {
					_ = callback(ChatEvent{Type: EventTypeThinking, Content: thinking.Thinking, IsEnd: false})
				}
				contentItem.OfThinking = &anthropic.ThinkingBlockParam{
					Thinking: thinking.Thinking,
				}
			case "tool_use":
				toolUse := msg.ContentBlock.AsToolUse()
				fmt.Println("Tool Use", "sessionId", a.session.ID, "agentName", a.name, "id", toolUse.ID, "tool", toolUse.Name, "input", string(toolUse.Input))
				toolUseInput.Reset()
				// Initialize tool use block
				contentItem.OfToolUse = &anthropic.ToolUseBlockParam{
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
					_ = callback(ChatEvent{Type: EventTypeThinking, Content: delta.Thinking, IsEnd: false})
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
					_ = callback(ChatEvent{Type: EventTypeText, Content: delta.Text, IsEnd: false})
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
			if callback != nil {
				_ = callback(ChatEvent{IsEnd: true})
			}
			result.OfAnthropic.Content = append(result.OfAnthropic.Content, contentItem)
			contentItem = anthropic.ContentBlockParamUnion{}
			fmt.Println("Content Block Stop", "sessionId", a.session.ID, "agentName", a.name, "index", msg.Index)
		}
	}

	return result, stopReason, nil
}

func (p *AnthropicProvider) ToolUse(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) ([]database.MessageUnion, error) {
	a := p.agent
	lastMessage := message.OfAnthropic
	result := database.MessageUnion{}
	if lastMessage == nil {
		return nil, fmt.Errorf("last message is not an Anthropic message")
	}
	// Find the tool use block
	toolUseBlocks := []anthropic.ToolUseBlockParam{}

	for _, block := range lastMessage.Content {
		if block.OfToolUse != nil {
			toolUseBlocks = append(toolUseBlocks, *block.OfToolUse)
		}
	}
	if len(toolUseBlocks) == 0 {
		fmt.Println("No tool use blocks found in chat history", "sessionId", a.session.ID, "agentName", a.name)
		return nil, fmt.Errorf("no tool use blocks found in chat history")
	}
	toolUseMessage := anthropic.MessageParam{
		Role:    anthropic.MessageParamRoleUser,
		Content: []anthropic.ContentBlockParamUnion{},
	}
	for _, toolUse := range toolUseBlocks {
		fmt.Println("Invoking tool", "name", toolUse.Name, "input", toolUse.Input)

		// Callback: Start tool execution
		if callback != nil {
			callback(ChatEvent{
				Type:    EventTypeToolUse,
				ToolUse: ToolCallWrapper{Anthropic: *lastMessage},
			})
		}

		// Normally toolUse.Name will have format <mcp>/<tool_name>
		parts := strings.SplitN(toolUse.Name, "--", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid tool name format, expected <mcp>--<tool_name>", "sessionId", a.session.ID, "agentName", a.name, "tool_name", toolUse.Name)
			return nil, fmt.Errorf("invalid tool name format, expected <mcp>--<tool_name>")
		}
		mcpName := parts[0]
		toolName := parts[1]
		mcpClient, ok := a.mcpClients[mcpName]
		if !ok {
			fmt.Println("MCP client not found", "sessionId", a.session.ID, "agentName", a.name, "mcpName", mcpName)
			return nil, fmt.Errorf("MCP client not found: %s", mcpName)
		}
		// Serialize the input JSON into map[string] any
		toolResponse, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: toolUse.Input,
				Meta: &mcp.Meta{
					AdditionalFields: map[string]any{
						"user_id":    a.session.CreatedBy.String(),
						"session_id": a.session.ID.String(),
					},
				},
			},
		})
		if err != nil {
			fmt.Println("Failed to call tool", "sessionId", a.session.ID, "agentName", a.name, "toolName", toolUse.Name, "error", err)
			return nil, fmt.Errorf("failed to call tool %s: %w", toolUse.Name, err)
		}

		toolResult := anthropic.ContentBlockParamUnion{
			OfToolResult: &anthropic.ToolResultBlockParam{
				ToolUseID: toolUse.ID,
				Content:   []anthropic.ToolResultBlockParamContentUnion{},
				IsError:   anthropic.Bool(toolResponse.IsError),
			},
		}
		// Convert the tool response content to anthropic format
		for _, content := range toolResponse.Content {
			switch content := content.(type) {
			case mcp.TextContent:
				toolResult.OfToolResult.Content = append(
					toolResult.OfToolResult.Content,
					anthropic.ToolResultBlockParamContentUnion{OfText: &anthropic.TextBlockParam{Text: content.Text}},
				)
				fmt.Println("Tool result: ", "sessionId", a.session.ID, "agentName", a.name, "tool_id", toolUse.ID, "tool_name", toolUse.Name, "text", content.Text)
			}
		}

		// Callback: End tool execution
		if callback != nil {
			callback(ChatEvent{
				Type:       EventTypeToolResult,
				ToolUse:    ToolCallWrapper{Anthropic: *lastMessage},
				ToolResult: ToolResultWrapper{Anthropic: toolResult},
			})
		}

		toolUseMessage.Content = append(toolUseMessage.Content, toolResult)
	}
	result.OfAnthropic = &toolUseMessage
	return []database.MessageUnion{result}, nil
}
