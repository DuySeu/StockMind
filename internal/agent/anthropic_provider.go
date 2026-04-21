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
		return database.StopReasonUnknown
	}
}

// Completion sends messages to the LLM and returns the response with streaming support
func (p *AnthropicProvider) Completion(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error) {
	if p.client == nil {
		return database.MessageUnion{}, database.StopReasonUnknown, fmt.Errorf("Anthropic client is not initialized")
	}

	if p.agent == nil {
		return database.MessageUnion{}, database.StopReasonUnknown, fmt.Errorf("agent is not initialized in AnthropicProvider")
	}

	// Prepare messages for Anthropic
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
	stream := p.client.Messages.NewStreaming(ctx, body)
	defer stream.Close()

	if err := stream.Err(); err != nil {
		return result, stopReason, err
	}

	var (
		contentBlockType string
		contentItem      anthropic.ContentBlockParamUnion
		toolUseInput     strings.Builder
		blockStarted     bool
		endEmitted       bool
	)

	for stream.Next() {
		chunk := stream.Current()

		switch chunk.Type {
		case "message_start":
			_ = chunk.AsMessageStart()

		case "content_block_start":
			ev := chunk.AsContentBlockStart()
			contentBlockType = ev.ContentBlock.Type
			contentItem = anthropic.ContentBlockParamUnion{}
			blockStarted = false

			switch ev.ContentBlock.Type {
			case "text":
				block := ev.ContentBlock.AsText()
				blockStarted = true
				contentItem.OfText = &anthropic.TextBlockParam{Text: block.Text}
			case "thinking":
				block := ev.ContentBlock.AsThinking()
				blockStarted = true
				contentItem.OfThinking = &anthropic.ThinkingBlockParam{
					Thinking:  block.Thinking,
					Signature: block.Signature,
				}
				if callback != nil {
					_ = callback(ChatEvent{
						Type:    EventTypeThinking,
						Content: block.Thinking,
					})
				}
			case "tool_use":
				toolUse := ev.ContentBlock.AsToolUse()
				blockStarted = true
				toolUseInput.Reset()
				if len(toolUse.Input) > 0 {
					_, _ = toolUseInput.Write(toolUse.Input)
				}
				param := toolUse.ToParam()
				contentItem.OfToolUse = &anthropic.ToolUseBlockParam{
					ID:           toolUse.ID,
					Name:         toolUse.Name,
					CacheControl: param.CacheControl,
					Input:        map[string]any{},
					Type:         param.Type,
				}
			}

		case "content_block_delta":
			if !blockStarted {
				continue
			}
			ev := chunk.AsContentBlockDelta()

			switch ev.Delta.Type {
			case "text_delta":
				delta := ev.Delta.AsTextDelta()
				if contentItem.OfText != nil {
					contentItem.OfText.Text += delta.Text
					if callback != nil {
						_ = callback(ChatEvent{
							Type:    EventTypeText,
							Content: delta.Text,
						})
					}
				}
			case "thinking_delta":
				delta := ev.Delta.AsThinkingDelta()
				if contentItem.OfThinking != nil {
					contentItem.OfThinking.Thinking += delta.Thinking
					if callback != nil {
						_ = callback(ChatEvent{
							Type:    EventTypeThinking,
							Content: delta.Thinking,
						})
					}
				}
			case "signature_delta":
				delta := ev.Delta.AsSignatureDelta()
				if contentItem.OfThinking != nil {
					contentItem.OfThinking.Signature += delta.Signature
				}
			case "input_json_delta":
				if contentBlockType == "tool_use" {
					delta := ev.Delta.AsInputJSONDelta()
					toolUseInput.WriteString(delta.PartialJSON)
				}
			}

		case "content_block_stop":
			if !blockStarted {
				contentBlockType = ""
				continue
			}

			if contentBlockType == "tool_use" && contentItem.OfToolUse != nil {
				raw := toolUseInput.String()
				if raw == "" {
					raw = "{}"
				}
				var input map[string]any
				if err := json.Unmarshal([]byte(raw), &input); err != nil {
					return result, stopReason, fmt.Errorf(
						"failed to unmarshal tool use input: %w", err)
				}
				contentItem.OfToolUse.Input = input
				toolUseInput.Reset()
			}

			result.OfAnthropic.Content = append(
				result.OfAnthropic.Content, contentItem)
			contentItem = anthropic.ContentBlockParamUnion{}
			contentBlockType = ""

		case "message_delta":
			ev := chunk.AsMessageDelta()
			stopReason = anthropicToDbStopReason(ev.Delta.StopReason)

		case "message_stop":
			if callback != nil {
				_ = callback(ChatEvent{IsEnd: true})
				endEmitted = true
			}
		}
	}

	if err := stream.Err(); err != nil {
		return result, stopReason, err
	}

	// Proxies or OpenRouter may omit message_stop; still close the SSE text block.
	if callback != nil && !endEmitted {
		_ = callback(ChatEvent{IsEnd: true})
	}

	return result, stopReason, nil
}

func (p *AnthropicProvider) ToolUse(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) ([]database.MessageUnion, error) {
	if p.agent == nil {
		return nil, fmt.Errorf("agent is not initialized")
	}
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
		return nil, fmt.Errorf("no tool use blocks found in chat history")
	}
	toolUseMessage := anthropic.MessageParam{
		Role:    anthropic.MessageParamRoleUser,
		Content: []anthropic.ContentBlockParamUnion{},
	}
	for _, toolUse := range toolUseBlocks {
		fmt.Printf("Invoking tool: name=%s, input=%v\n", toolUse.Name, toolUse.Input)

		// Callback: Start tool execution
		if callback != nil {
			_ = callback(ChatEvent{
				Type:    EventTypeToolUse,
				ToolUse: ToolCallWrapper{Anthropic: *lastMessage},
			})
		}

		// Tool name format <mcp>--<tool_name>
		parts := strings.SplitN(toolUse.Name, "--", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tool name format: %s", toolUse.Name)
		}
		mcpName := parts[0]
		toolName := parts[1]
		mcpClient, ok := a.mcpClients[mcpName]
		if !ok {
			return nil, fmt.Errorf("MCP client not found: %s", mcpName)
		}

		// Call tool with metadata
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
			return nil, fmt.Errorf("failed to call tool %s: %w", toolUse.Name, err)
		}

		toolResult := anthropic.ContentBlockParamUnion{
			OfToolResult: &anthropic.ToolResultBlockParam{
				ToolUseID: toolUse.ID,
				Content:   []anthropic.ToolResultBlockParamContentUnion{},
				IsError:   anthropic.Bool(toolResponse.IsError),
			},
		}

		// Convert results back to SDK format
		for _, content := range toolResponse.Content {
			if tc, ok := content.(mcp.TextContent); ok {
				toolResult.OfToolResult.Content = append(
					toolResult.OfToolResult.Content,
					anthropic.ToolResultBlockParamContentUnion{OfText: &anthropic.TextBlockParam{Text: tc.Text}},
				)
			}
		}

		// Callback: End tool execution
		if callback != nil {
			_ = callback(ChatEvent{
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
