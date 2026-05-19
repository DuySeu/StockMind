package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/tools"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// NewOpenAIClient builds an OpenAI-compatible client from the given config.
func NewOpenAIClient(config common.OpenAI) (*openai.Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not found")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
		option.WithHTTPClient(common.SharedHTTPClient),
	}

	if strings.Contains(config.BaseURL, "openrouter.ai") {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	} else if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	client := openai.NewClient(opts...)
	return &client, nil
}

// emitTextDeltas forwards visible assistant text from a stream delta directly
// onto ch.
func emitTextDeltas(ch chan<- database.StreamEvent, delta openai.ChatCompletionChunkChoiceDelta) {
	if delta.Content != "" {
		ch <- database.StreamEvent{Type: database.EventText, Content: delta.Content}
	}
	if delta.Refusal != "" {
		ch <- database.StreamEvent{Type: database.EventText, Content: delta.Refusal}
	}
}

// OpenAICompletion sends messages to an OpenAI-compatible endpoint and returns a streaming event channel.
func OpenAICompletion(client *openai.Client, model string, ctx context.Context, messages []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
	ch := make(chan database.StreamEvent, 256)

	openaiMsgs := mapToOpenAIMessages(messages)

	var reqTools []openai.ChatCompletionToolUnionParam
	for _, t := range tools {
		reqTools = append(reqTools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  t.Schema,
		}))
	}

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: openaiMsgs,
	}
	if len(reqTools) > 0 {
		params.Tools = reqTools
	}

	stream := client.Chat.Completions.NewStreaming(ctx, params)

	go func() {
		defer close(ch)

		// Accumulate parallel tool calls by stream index.
		type toolCallAccum struct {
			ID        string
			Name      string
			Arguments string
		}
		pendingToolCalls := make(map[int]*toolCallAccum)

		for stream.Next() {
			chunk := stream.Current()

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			delta := choice.Delta

			emitTextDeltas(ch, delta)

			for _, tc := range delta.ToolCalls {
				idx := int(tc.Index)
				existing, ok := pendingToolCalls[idx]
				if !ok {
					existing = &toolCallAccum{}
					pendingToolCalls[idx] = existing
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Name = tc.Function.Name
				}
				existing.Arguments += tc.Function.Arguments
			}

			if choice.FinishReason == "tool_calls" {
				indices := make([]int, 0, len(pendingToolCalls))
				for k := range pendingToolCalls {
					indices = append(indices, k)
				}
				sort.Ints(indices)
				for _, k := range indices {
					full := pendingToolCalls[k]
					if full == nil || full.Name == "" {
						continue
					}
					ch <- database.StreamEvent{
						Type: database.EventToolCall,
						Data: database.Tool{
							ID:        full.ID,
							Name:      full.Name,
							Arguments: full.Arguments,
						},
					}
				}
				pendingToolCalls = make(map[int]*toolCallAccum)
			}
		}

		if err := stream.Err(); err != nil {
			ch <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
			return
		}

		ch <- database.StreamEvent{Type: database.EventDone}
	}()

	return ch, nil
}

func mapToOpenAIMessages(messages []database.Message) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion

	for _, m := range messages {
		content := strings.TrimSpace(m.Content)

		switch m.Role {
		case "system":
			result = append(result, openai.SystemMessage(content))
		case "user":
			result = append(result, openai.UserMessage(content))
		case "assistant":
			msg := openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(content),
				},
			}

			var toolResults []openai.ChatCompletionMessageParamUnion

			if len(m.Metadata) > 0 {
				meta := m.Metadata[0]
				for _, t := range meta.Tool {
					if t.ID != "" {
						msg.ToolCalls = append(msg.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
								ID: t.ID,
								Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      t.Name,
									Arguments: t.Arguments,
								},
							},
						})

						toolResults = append(toolResults, openai.ToolMessage(t.Result, t.ID))
					}
				}
			}

			result = append(result, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &msg,
			})
			result = append(result, toolResults...)
		default:
			result = append(result, openai.UserMessage(content))
		}
	}

	return result
}
