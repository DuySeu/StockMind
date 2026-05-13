package core

import (
	"context"
	"fmt"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/mark3labs/mcp-go/mcp"
)

// NewOpenRouterClient builds an OpenRouter client from the given config.
func NewOpenRouterClient(config common.OpenRouter) (*openrouter.OpenRouter, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is not found")
	}
	client := openrouter.New(
		openrouter.WithSecurity(config.APIKey),
		openrouter.WithClient(common.SharedHTTPClient),
	)
	return client, nil
}

// OpenRouterCompletion calls the OpenRouter API using the official SDK.
func OpenRouterCompletion(client *openrouter.OpenRouter, model string, ctx context.Context, messages []database.Message, tools []mcp.Tool) (<-chan common.StreamEvent, error) {
	ch := make(chan common.StreamEvent, 256)

	mapped := mapToOpenRouterChat(messages, tools)

	req := components.ChatRequest{
		Model:    openrouter.Pointer(model),
		Messages: mapped.Messages,
		Stream:   openrouter.Pointer(true),
	}
	if len(mapped.Tools) > 0 {
		req.Tools = mapped.Tools
	}

	res, err := client.Chat.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: send request: %w", err)
	}

	if res.EventStream == nil {
		return nil, fmt.Errorf("openrouter: expected event stream but got none")
	}

	go func() {
		defer close(ch)
		defer res.EventStream.Close()

		// Accumulate tool call arguments across delta chunks
		pendingToolCalls := map[int]*database.Tool{}

		for res.EventStream.Next() {
			event := res.EventStream.Value()
			chunk := event.Data

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]

			// ── Text delta ───────────────────────────────────────────────────
			// OptionalNullable is map[bool]*T keyed by true (see go-sdk/optionalnullable).
			if ptr, set := choice.Delta.Content.Get(); set && ptr != nil && *ptr != "" {
				ch <- common.StreamEvent{Type: common.EventText, Content: *ptr}
			}
			// Reasoning (a.k.a. "thinking") deltas are streamed separately so
			// clients can render them as a distinct thought stream.
			if ptr, set := choice.Delta.Reasoning.Get(); set && ptr != nil && *ptr != "" {
				ch <- common.StreamEvent{Type: common.EventThinking, Content: *ptr}
			}

			// ── Tool call deltas (accumulate by index) ───────────────────────
			for _, tc := range choice.Delta.ToolCalls {
				idx := int(tc.Index)
				if existing, ok := pendingToolCalls[idx]; ok {
					// Append argument fragment
					if tc.Function != nil && tc.Function.Arguments != nil {
						existing.Arguments += *tc.Function.Arguments
					}
				} else {
					// First delta for this tool call
					tool := &database.Tool{}
					if tc.ID != nil {
						tool.ID = *tc.ID
					}
					if tc.Function != nil {
						if tc.Function.Name != nil {
							tool.Name = *tc.Function.Name
						}
						if tc.Function.Arguments != nil {
							tool.Arguments = *tc.Function.Arguments
						}
					}
					pendingToolCalls[idx] = tool
				}
			}

			// ── Finish reasons ───────────────────────────────────────────────
			if choice.FinishReason != nil {
				reason := string(*choice.FinishReason)
				switch reason {
				case "tool_calls":
					// Emit each fully-assembled tool call
					for i := 0; i < len(pendingToolCalls); i++ {
						if tc, ok := pendingToolCalls[i]; ok {
							ch <- common.StreamEvent{
								Type: common.EventToolCall,
								Data: *tc,
							}
						}
					}
					// Reset for a potential second tool-call round
					pendingToolCalls = map[int]*database.Tool{}

				case "stop":
					ch <- common.StreamEvent{Type: common.EventDone}
					return
				}
			}
		}

		if err := res.EventStream.Err(); err != nil {
			ch <- common.StreamEvent{Type: common.EventError, Data: err.Error()}
			return
		}
		ch <- common.StreamEvent{Type: common.EventDone}
	}()

	return ch, nil
}

// openRouterChatMapped is the OpenRouter Chat request payload derived from domain types.
type openRouterChatMapped struct {
	Messages []components.ChatMessages
	Tools    []components.ChatFunctionTool
}

// mapToOpenRouterChat maps conversation history and MCP tool definitions into OpenRouter API
func mapToOpenRouterChat(messages []database.Message, tools []mcp.Tool) openRouterChatMapped {
	out := openRouterChatMapped{
		Messages: make([]components.ChatMessages, 0, len(messages)),
	}

	appendToolResultMessages := func(meta []database.Metadata) {
		if len(meta) == 0 {
			return
		}
		for _, t := range meta[0].Tool {
			if t.ID == "" {
				continue
			}
			out.Messages = append(out.Messages, components.CreateChatMessagesTool(components.ChatToolMessage{
				Role:       components.ChatToolMessageRoleTool,
				ToolCallID: t.ID,
				Content:    components.CreateChatToolMessageContentStr(t.Output),
			}))
		}
	}

	for _, m := range messages {
		var msg components.ChatMessages
		role := strings.ToLower(m.Role)
		content := strings.TrimSpace(m.Content)

		switch role {
		case "system":
			msg = components.CreateChatMessagesSystem(components.ChatSystemMessage{
				Role:    components.ChatSystemMessageRoleSystem,
				Content: components.CreateChatSystemMessageContentStr(content),
			})
		case "user":
			msg = components.CreateChatMessagesUser(components.ChatUserMessage{
				Role:    components.ChatUserMessageRoleUser,
				Content: components.CreateChatUserMessageContentStr(content),
			})
		case "assistant":
			var toolCalls []components.ChatToolCall
			if len(m.Metadata) > 0 {
				for _, t := range m.Metadata[0].Tool {
					if t.ID != "" {
						toolCalls = append(toolCalls, components.ChatToolCall{
							ID:   t.ID,
							Type: components.ChatToolCallTypeFunction,
							Function: components.ChatToolCallFunction{
								Name:      t.Name,
								Arguments: t.Arguments,
							},
						})
					}
				}
			}
			assistantMsg := components.ChatAssistantMessage{
				Role:      components.ChatAssistantMessageRoleAssistant,
				ToolCalls: toolCalls,
			}
			if content != "" {
				assistantMsg.Content = optionalnullable.From(openrouter.Pointer(components.CreateChatAssistantMessageContentStr(content)))
			}
			out.Messages = append(out.Messages, components.CreateChatMessagesAssistant(assistantMsg))
			appendToolResultMessages(m.Metadata)
			continue
		case "tool":
			appendToolResultMessages(m.Metadata)
			continue
		}

		out.Messages = append(out.Messages, msg)
	}

	if len(tools) > 0 {
		out.Tools = make([]components.ChatFunctionTool, 0, len(tools))
		for _, t := range tools {
			out.Tools = append(out.Tools, components.CreateChatFunctionToolChatFunctionToolFunction(components.ChatFunctionToolFunction{
				Type: components.ChatFunctionToolTypeFunction,
				Function: components.ChatFunctionToolFunctionFunction{
					Name:        t.Name,
					Description: openrouter.String(t.Description),
					Parameters:  schemaToMap(t.InputSchema),
				},
			}))
		}
	}
	return out
}

// OpenRouterEmbedding calls the OpenRouter embeddings endpoint using the shared SDK client.
// It accepts a batch of strings and returns float32 vectors.
func OpenRouterEmbedding(ctx context.Context, client *openrouter.OpenRouter, model string, inputs []string) ([][]float32, error) {
	req := operations.CreateEmbeddingsRequest{
		Model: model,
		Input: operations.CreateInputUnionArrayOfStr(inputs),
	}

	resp, err := client.Embeddings.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openrouter embedding: %w", err)
	}
	if resp == nil || resp.CreateEmbeddingsResponseBody == nil {
		return nil, fmt.Errorf("openrouter embedding: empty response")
	}

	data := resp.CreateEmbeddingsResponseBody.Data
	vecs := make([][]float32, len(data))
	for i, d := range data {
		if d.Embedding.ArrayOfNumber == nil {
			return nil, fmt.Errorf("openrouter embedding: unexpected format at index %d", i)
		}
		v := make([]float32, len(d.Embedding.ArrayOfNumber))
		for j, f := range d.Embedding.ArrayOfNumber {
			v[j] = float32(f)
		}
		vecs[i] = v
	}
	return vecs, nil
}

// schemaToMap converts an MCP input schema into the generic map shape expected
// by the OpenRouter SDK without going through json.Marshal/Unmarshal.
func schemaToMap(schema mcp.ToolInputSchema) map[string]any {
	m := make(map[string]any, 5)
	if schema.Type != "" {
		m["type"] = schema.Type
	}
	if schema.Properties != nil {
		m["properties"] = schema.Properties
	} else {
		m["properties"] = map[string]any{}
	}
	if len(schema.Required) > 0 {
		m["required"] = schema.Required
	}
	if schema.AdditionalProperties != nil {
		m["additionalProperties"] = schema.AdditionalProperties
	}
	if schema.Defs != nil {
		m["$defs"] = schema.Defs
	}
	return m
}
