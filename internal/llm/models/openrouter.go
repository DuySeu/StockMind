package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/tools"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
)

// OpenRouterCompletionParams carries the per-call inputs shared by both OpenRouter entry points.
type OpenRouterCompletionParams struct {
	Context context.Context
	Client  *openrouter.OpenRouter
	Model   string
	Prompt  string
}

// openRouterProvider adapts the OpenRouter SDK glue to the Provider interface.
type openRouterProvider struct {
	client *openrouter.OpenRouter
	model  string
}

// Completion streams a chat completion for the conversation history.
func (p *openRouterProvider) Completion(ctx context.Context, history []database.Message, toolDefs []*tools.Tool, systemPrompt string) (<-chan database.StreamEvent, error) {
	return OpenRouterCompletion(OpenRouterCompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  systemPrompt,
	}, history, toolDefs)
}

// StructuredCompletion requests a single JSON completion and unmarshals it into result.
func (p *openRouterProvider) StructuredCompletion(ctx context.Context, prompt string, result any) error {
	return OpenRouterStructuredCompletion(OpenRouterCompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  prompt,
	}, result)
}

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
func OpenRouterCompletion(params OpenRouterCompletionParams, messages []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
	ch := make(chan database.StreamEvent, 256)
	msgs := []components.ChatMessages{}
	// -- Map messages --
	openrouterMsgs, err := DBToOpenRouterMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("openrouter: failed to map messages: %w", err)
	}

	// Prepend the system prompt before building the request so it is actually
	// included in req.Messages (must happen before the append below).
	if params.Prompt != "" {
		msgs = append(msgs, components.CreateChatMessagesSystem(
			components.ChatSystemMessage{
				Content: components.CreateChatSystemMessageContentStr(params.Prompt),
				Role:    components.ChatSystemMessageRoleSystem,
			},
		))
	}

	req := components.ChatRequest{
		Model:    openrouter.Pointer(params.Model),
		Messages: append(msgs, openrouterMsgs...),
		Stream:   openrouter.Pointer(true),
	}
	if len(tools) > 0 {
		req.Tools = make([]components.ChatFunctionTool, 0, len(tools))
		for _, t := range tools {
			req.Tools = append(req.Tools, components.CreateChatFunctionToolChatFunctionToolFunction(components.ChatFunctionToolFunction{
				Type: components.ChatFunctionToolTypeFunction,
				Function: components.ChatFunctionToolFunctionFunction{
					Name:        t.Name,
					Description: openrouter.String(t.Description),
					Parameters:  t.Schema,
				},
			}))
		}
	}

	res, err := params.Client.Chat.Send(params.Context, req)
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
		var contentLen, reasoningLen int // diagnostic: bytes seen this stream

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
				contentLen += len(*ptr)
				ch <- database.StreamEvent{Type: database.EventText, Content: *ptr}
			}
			// Reasoning (a.k.a. "thinking") deltas are streamed separately so
			// clients can render them as a distinct thought stream.
			if ptr, set := choice.Delta.Reasoning.Get(); set && ptr != nil && *ptr != "" {
				reasoningLen += len(*ptr)
				ch <- database.StreamEvent{Type: database.EventThinking, Content: *ptr}
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
				slog.Info("openrouter: finish_reason", "reason", reason,
					"content_len", contentLen, "reasoning_len", reasoningLen,
					"pending_tool_calls", len(pendingToolCalls))
				switch reason {
				case "tool_calls":
					// Emit each fully-assembled tool call in index order.
					// Don't assume indices are contiguous 0..n-1 — collect and sort keys.
					indices := make([]int, 0, len(pendingToolCalls))
					for k := range pendingToolCalls {
						indices = append(indices, k)
					}
					sort.Ints(indices)
					for _, k := range indices {
						ch <- database.StreamEvent{
							Type: database.EventToolCall,
							Data: *pendingToolCalls[k],
						}
					}
					// Reset for a potential second tool-call round
					pendingToolCalls = map[int]*database.Tool{}

				case "stop":
					ch <- database.StreamEvent{Type: database.EventDone}
					return
				}
			}
		}

		if err := res.EventStream.Err(); err != nil {
			slog.Error("openrouter: stream error", "error", err,
				"content_len", contentLen, "reasoning_len", reasoningLen)
			ch <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
			return
		}
		slog.Info("openrouter: stream end (no stop finish_reason)",
			"content_len", contentLen, "reasoning_len", reasoningLen,
			"pending_tool_calls", len(pendingToolCalls))
		ch <- database.StreamEvent{Type: database.EventDone}
	}()

	return ch, nil
}

// OpenRouterStructuredCompletion calls the OpenRouter API without streaming using
// response_format json_object. The prompt must describe the desired JSON structure.
// Result must be a pointer; the JSON response is unmarshalled into it.
func OpenRouterStructuredCompletion(params OpenRouterCompletionParams, result any) error {
	rf := components.CreateResponseFormatJSONObject(components.FormatJSONObjectConfig{})

	req := components.ChatRequest{
		Model: openrouter.Pointer(params.Model),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesUser(components.ChatUserMessage{
				Content: components.CreateChatUserMessageContentStr(params.Prompt),
				Role:    components.ChatUserMessageRoleUser,
			}),
		},
		Stream:         openrouter.Pointer(false),
		ResponseFormat: &rf,
	}

	res, err := params.Client.Chat.Send(params.Context, req)
	if err != nil {
		return fmt.Errorf("openrouter structured: %w", err)
	}
	if res.ChatResult == nil || len(res.ChatResult.Choices) == 0 {
		return fmt.Errorf("openrouter structured: empty response")
	}

	ptr, set := res.ChatResult.Choices[0].Message.Content.Get()
	if !set || ptr == nil || ptr.Str == nil {
		return fmt.Errorf("openrouter structured: no content")
	}
	if err := json.Unmarshal([]byte(*ptr.Str), result); err != nil {
		return fmt.Errorf("openrouter structured: %w", err)
	}
	return nil
}
