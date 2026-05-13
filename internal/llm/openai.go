package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"

	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"
)

// NewOpenAIClient builds an OpenAI-compatible client from the given config.
// Both branches reuse sharedHTTPClient so connections are pooled across calls.
func NewOpenAIClient(config common.OpenAI) (*openai.Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not found")
	}

	defaultConfig := openai.DefaultConfig(config.APIKey)
	defaultConfig.HTTPClient = common.SharedHTTPClient

	if strings.Contains(config.BaseURL, "openrouter.ai") {
		defaultConfig.BaseURL = config.BaseURL
		// OpenRouter sends frequent empty keep-alive chunks which can exceed the default limit
		defaultConfig.EmptyMessagesLimit = 300000
	}

	return openai.NewClientWithConfig(defaultConfig), nil
}

// emitTextDeltas forwards visible assistant text from a stream delta directly
// onto ch. Some providers put tokens in reasoning_content (e.g. DeepSeek-R1)
// or refusal. Pushing directly avoids allocating a slice per delta.
func emitTextDeltas(ch chan<- common.StreamEvent, delta openai.ChatCompletionStreamChoiceDelta) {
	if delta.Content != "" {
		ch <- common.StreamEvent{Type: common.EventText, Content: delta.Content}
	}
	if delta.ReasoningContent != "" {
		ch <- common.StreamEvent{Type: common.EventText, Content: delta.ReasoningContent}
	}
	if delta.Refusal != "" {
		ch <- common.StreamEvent{Type: common.EventText, Content: delta.Refusal}
	}
}

// OpenAICompletion sends messages to an OpenAI-compatible endpoint and returns a streaming event channel.
func OpenAICompletion(client *openai.Client, model string, ctx context.Context, messages []database.Message, tools []mcp.Tool) (<-chan common.StreamEvent, error) {
	ch := make(chan common.StreamEvent, 256)

	openaiMsgs := mapToOpenAIMessages(messages)

	reqTools := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		reqTools = append(reqTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMsgs,
		Stream:   true,
	}
	if len(reqTools) > 0 {
		req.Tools = reqTools
	}

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai stream: %w", err)
	}

	go func() {
		defer close(ch)
		defer stream.Close()

		// Accumulate parallel tool calls by stream index (see ToolCall.Index in go-openai).
		pendingToolCalls := make(map[int]*openai.ToolCall)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				ch <- common.StreamEvent{Type: common.EventDone}
				return
			}
			if err != nil {
				ch <- common.StreamEvent{Type: common.EventError, Data: err.Error()}
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta

			emitTextDeltas(ch, delta)

			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}
				existing, ok := pendingToolCalls[idx]
				if !ok {
					existing = &openai.ToolCall{
						Type: openai.ToolTypeFunction,
					}
					pendingToolCalls[idx] = existing
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Function.Name = tc.Function.Name
				}
				existing.Function.Arguments += tc.Function.Arguments
			}

			if response.Choices[0].FinishReason == openai.FinishReasonToolCalls {
				indices := make([]int, 0, len(pendingToolCalls))
				for k := range pendingToolCalls {
					indices = append(indices, k)
				}
				sort.Ints(indices)
				for _, k := range indices {
					full := pendingToolCalls[k]
					if full == nil || full.Function.Name == "" {
						continue
					}
					ch <- common.StreamEvent{
						Type: common.EventToolCall,
						Data: database.Tool{
							ID:        full.ID,
							Name:      full.Function.Name,
							Arguments: full.Function.Arguments,
						},
					}
				}
				pendingToolCalls = make(map[int]*openai.ToolCall)
			}
		}
	}()

	return ch, nil
}

func mapToOpenAIMessages(messages []database.Message) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage

	for _, m := range messages {
		res := openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: strings.TrimSpace(m.Content),
		}

		var toolResults []openai.ChatCompletionMessage

		if len(m.Metadata) > 0 {
			meta := m.Metadata[0]
			for _, t := range meta.Tool {
				if t.ID != "" {
					res.ToolCalls = append(res.ToolCalls, openai.ToolCall{
						ID:   t.ID,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      t.Name,
							Arguments: t.Arguments,
						},
					})

					// OpenAI treats tool_result as a separate message entirely with role=tool
					toolResults = append(toolResults, openai.ChatCompletionMessage{
						Role:       string(openai.ChatMessageRoleTool),
						ToolCallID: t.ID,
						Content:    t.Output,
					})
				}
			}
		}

		result = append(result, res)
		result = append(result, toolResults...)
	}

	return result
}
