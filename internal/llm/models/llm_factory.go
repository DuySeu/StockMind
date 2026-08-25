package models

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/tools"
)

// Provider is the provider-agnostic contract the LLM service depends on.
// Each concrete provider (OpenAI, Anthropic, OpenRouter) implements it, keeping
// SDK-specific glue out of the service and behind one uniform surface.
type Provider interface {
	// Completion streams one assistant turn (text deltas + tool calls) as StreamEvents.
	Completion(ctx context.Context, history []database.Message, toolDefs []*tools.Tool, systemPrompt string) (<-chan database.StreamEvent, error)
	// StructuredCompletion performs a non-streaming completion whose JSON output
	// is unmarshalled into result (which must be a pointer).
	StructuredCompletion(ctx context.Context, prompt string, result any) error
}

// NewProvider constructs the Provider selected by cfg (via LLM_PROVIDER / LLM_MODEL).
// This is the single place where a provider is chosen and its client is built.
// To add a provider: implement Provider in its own file and add one case here.
func NewProvider(ctx context.Context, cfg common.LLM) (Provider, error) {
	model := cfg.GetLLMModelName()
	switch cfg.GetProviderName() {
	case database.ModelProviderOpenAI:
		client, err := NewOpenAIClient(cfg.OpenAI)
		if err != nil {
			return nil, err
		}
		return &openAIProvider{client: client, model: model}, nil
	case database.ModelProviderAnthropic:
		client, err := NewAnthropicClient(ctx, cfg.Anthropic)
		if err != nil {
			return nil, err
		}
		return &anthropicProvider{client: client, model: model}, nil
	case database.ModelProviderOpenRouter:
		client, err := NewOpenRouterClient(cfg.OpenRouter)
		if err != nil {
			return nil, err
		}
		return &openRouterProvider{client: client, model: model}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %q", cfg.GetProviderName())
	}
}
