package core

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/providers"
	"stockmind/internal/llm/tools"
)

type completionFunc func(context.Context, []database.Message, []*tools.Tool, string) (<-chan database.StreamEvent, error)
type structuredCompletionFunc func(ctx context.Context, prompt string, result any) error

// LLMService handles LLM interactions (agentic tool loop + summarization).
// It has no database dependency — all persistence is handled by the caller.
type LLMService struct {
	tools                *tools.Manager
	completion           completionFunc
	structuredCompletion structuredCompletionFunc
}

func NewLLMService(ctx context.Context, cfg common.LLM, toolMgr *tools.Manager) (*LLMService, error) {
	var completion completionFunc
	var structuredCompletion structuredCompletionFunc
	switch cfg.GetProviderName() {
	case database.ModelProviderOpenAI:
		client, err := providers.NewOpenAIClient(cfg.OpenAI)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool, prompt string) (<-chan database.StreamEvent, error) {
			return providers.OpenAICompletion(providers.OpenAICompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
			}, history, tools)
		}
		structuredCompletion = func(ctx context.Context, prompt string, result any) error {
			return providers.OpenAIStructuredCompletion(providers.OpenAICompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
				Prompt:  prompt,
			}, result)
		}
	case database.ModelProviderAnthropic:
		client, err := providers.NewAnthropicClient(ctx, cfg.Anthropic)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool, prompt string) (<-chan database.StreamEvent, error) {
			return providers.AnthropicCompletion(providers.AnthropicCompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
				Prompt:  prompt,
			}, history, tools)
		}
		structuredCompletion = func(ctx context.Context, prompt string, result any) error {
			return providers.AnthropicStructuredCompletion(providers.AnthropicCompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
				Prompt:  prompt,
			}, result)
		}
	case database.ModelProviderOpenRouter:
		client, err := providers.NewOpenRouterClient(cfg.OpenRouter)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool, prompt string) (<-chan database.StreamEvent, error) {
			return providers.OpenRouterCompletion(providers.OpenRouterCompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
				Prompt:  prompt,
			}, history, tools)
		}
		structuredCompletion = func(ctx context.Context, prompt string, result any) error {
			return providers.OpenRouterStructuredCompletion(providers.OpenRouterCompletionParams{
				Context: ctx,
				Client:  client,
				Model:   cfg.GetLLMModelName(),
				Prompt:  prompt,
			}, result)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %q", cfg.GetProviderName())
	}

	return &LLMService{
		tools:                toolMgr,
		completion:           completion,
		structuredCompletion: structuredCompletion,
	}, nil
}

// ──────── Agentic Tool Loop ────────

// runToolRound runs each tool in order, streams ToolResult events, and appends
// one assistant history row containing every tool call/output pair.
func (s *LLMService) runToolRound(ctx context.Context, outputCh chan<- database.StreamEvent, history []database.Message, pending []database.Tool) []database.Message {
	assembled := make([]database.Tool, 0, len(pending))
	for _, tc := range pending {
		result, execErr := s.tools.Execute(ctx, tc.Name, tc.Arguments)
		isError := "false"
		if execErr != nil {
			result = execErr.Error()
			isError = "true"
		}
		tr := database.Tool{ID: tc.ID, Result: result, IsError: isError}
		outputCh <- database.StreamEvent{Type: database.EventToolResult, Data: tr}
		assembled = append(assembled, database.Tool{
			ID:      tc.ID,
			Result:  tr.Result,
			IsError: tr.IsError,
		})
	}
	meta := []database.Metadata{{Tool: assembled}}
	return append(history, database.Message{Role: "assistant", Metadata: meta})
}

// LLMOptions holds optional parameters for LLMChat.
type LLMOptions struct {
	SystemPrompt string
}

// LLMChat runs the agentic tool loop: call LLM → execute tool calls → append results → repeat until done.
// It has no database interactions or session management.
func (s *LLMService) LLMChat(ctx context.Context, history []database.Message, opts ...LLMOptions) (<-chan database.StreamEvent, error) {
	outputCh := make(chan database.StreamEvent, 4)

	go func() {
		defer close(outputCh)
		toolDefs := s.tools.All()

	nextProviderRound:
		for {
			streamCh, err := s.completion(ctx, history, toolDefs, opts[0].SystemPrompt)
			if err != nil {
				outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
				return
			}

			var pendingTools []database.Tool

			for event := range streamCh {
				switch event.Type {
				case database.EventText:
					outputCh <- event

				case database.EventToolCall:
					tc := event.Data.(database.Tool)
					outputCh <- event
					pendingTools = append(pendingTools, tc)

				case database.EventDone:
					if len(pendingTools) > 0 {
						history = s.runToolRound(ctx, outputCh, history, pendingTools)
						pendingTools = pendingTools[:0]
						goto nextProviderRound
					}
					outputCh <- event
					return

				case database.EventError:
					outputCh <- event
					return
				}
			}

			if len(pendingTools) > 0 {
				history = s.runToolRound(ctx, outputCh, history, pendingTools)
				continue
			}
			return
		}
	}()

	return outputCh, nil
}
