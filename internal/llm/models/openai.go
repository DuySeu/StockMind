package models

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/tools"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAICompletionParams carries the per-call inputs shared by both OpenAI entry points.
type OpenAICompletionParams struct {
	Context context.Context
	Client  *openai.Client
	Model   string
	Prompt  string
}

// openAIProvider adapts the OpenAI SDK glue to the Provider interface.
type openAIProvider struct {
	client *openai.Client
	model  string
}

// Completion streams a chat completion for the conversation history.
func (p *openAIProvider) Completion(ctx context.Context, history []database.Message, toolDefs []*tools.Tool, systemPrompt string) (<-chan database.StreamEvent, error) {
	return OpenAICompletion(OpenAICompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  systemPrompt,
	}, history, toolDefs)
}

// StructuredCompletion requests a single JSON completion and unmarshals it into result.
func (p *openAIProvider) StructuredCompletion(ctx context.Context, prompt string, result any) error {
	return OpenAIStructuredCompletion(OpenAICompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  prompt,
	}, result)
}

// NewOpenAIClient builds an OpenAI-compatible client from the given config.
func NewOpenAIClient(config common.OpenAI) (*openai.Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not found")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
		option.WithHTTPClient(common.SharedHTTPClient),
	}

	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	client := openai.NewClient(opts...)
	return &client, nil
}

// OpenAICompletion sends messages to an OpenAI-compatible endpoint and returns a streaming event channel.
func OpenAICompletion(params OpenAICompletionParams, messages []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
	ch := make(chan database.StreamEvent, 256)
	msgs := []openai.ChatCompletionMessageParamUnion{}

	openaiMsgs, err := DBToOpenAIMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to map messages: %w", err)
	}
	if params.Prompt != "" {
		msgs = append(msgs, openai.SystemMessage(params.Prompt))
	}

	var reqTools []openai.ChatCompletionToolUnionParam
	for _, t := range tools {
		reqTools = append(reqTools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  t.Schema,
		}))
	}

	req := openai.ChatCompletionNewParams{
		Model:    params.Model,
		Messages: append(msgs, openaiMsgs...),
	}
	if len(reqTools) > 0 {
		req.Tools = reqTools
	}

	stream := params.Client.Chat.Completions.NewStreaming(params.Context, req)

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

			// Forward visible assistant text.
			if delta.Content != "" {
				ch <- database.StreamEvent{Type: database.EventText, Content: delta.Content}
			}
			if delta.Refusal != "" {
				ch <- database.StreamEvent{Type: database.EventText, Content: delta.Refusal}
			}

			// Accumulate tool call fragments by stream index.
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

// OpenAIStructuredCompletion calls the OpenAI API without streaming using
// response_format json_object. The prompt must describe the desired JSON structure.
// Result must be a pointer; the JSON response is unmarshalled into it.
func OpenAIStructuredCompletion(params OpenAICompletionParams, result any) error {
	req := openai.ChatCompletionNewParams{
		Model: params.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(params.Prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		},
	}

	res, err := params.Client.Chat.Completions.New(params.Context, req)
	if err != nil {
		return fmt.Errorf("openai structured: %w", err)
	}
	if len(res.Choices) == 0 {
		return fmt.Errorf("openai structured: empty response")
	}
	if err := json.Unmarshal([]byte(res.Choices[0].Message.Content), result); err != nil {
		return fmt.Errorf("openai structured: %w", err)
	}
	return nil
}
