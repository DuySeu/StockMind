package knowledge_base

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultBatchSize = 20
)

// embedFunc is the low-level function that calls the embedding API.
type embedFunc func(ctx context.Context, inputs []string) ([][]float32, error)

// EmbedService provides embedding generation, abstracting away the provider.
// Provider is selected once at init time via NewEmbedService.
type EmbedService struct {
	embed      embedFunc
	model      string
	dimensions int
	batchSize  int
}

// NewEmbedService creates an EmbedService for the given provider and model.
// Supported providers: "openrouter", "openai".
func NewEmbedService(provider database.ModelProvider, model string, dimensions int, cfg common.LLMProvider) (*EmbedService, error) {
	if dimensions <= 0 {
		dimensions = embeddingDimensions
	}

	var fn embedFunc
	switch provider {
	case database.ModelProviderOpenRouter:
		if cfg.OpenRouter.APIKey == "" {
			return nil, fmt.Errorf("embed_service: OPENROUTER_API_KEY is required")
		}
		client := openrouter.New(
			openrouter.WithSecurity(cfg.OpenRouter.APIKey),
			openrouter.WithClient(common.SharedHTTPClient),
		)
		fn = openRouterEmbed(client, model)

	case database.ModelProviderOpenAI:
		if cfg.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("embed_service: OpenAI API key is required")
		}
		oaiCfg := openai.DefaultConfig(cfg.OpenAI.APIKey)
		oaiCfg.HTTPClient = common.SharedHTTPClient
		if cfg.OpenAI.BaseURL != "" {
			oaiCfg.BaseURL = cfg.OpenAI.BaseURL
		}
		client := openai.NewClientWithConfig(oaiCfg)
		fn = openAIEmbed(client, model)

	default:
		return nil, fmt.Errorf("embed_service: unsupported provider %q", provider)
	}

	return &EmbedService{
		embed:      fn,
		model:      model,
		dimensions: dimensions,
		batchSize:  defaultBatchSize,
	}, nil
}

func (s *EmbedService) Dimensions() int { return s.dimensions }

func (s *EmbedService) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("embed_service: at least one input is required")
	}

	results := make([][]float32, len(inputs))
	for start := 0; start < len(inputs); start += s.batchSize {
		end := start + s.batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		vecs, err := s.embed(ctx, inputs[start:end])
		if err != nil {
			return nil, fmt.Errorf("embed_service: batch [%d:%d] failed: %w", start, end, err)
		}
		copy(results[start:end], vecs)
	}
	return results, nil
}

func (s *EmbedService) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("embed_service: query must not be empty")
	}
	res, err := s.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("embed_service: no embedding returned")
	}
	return res[0], nil
}

// ---- Provider implementations ----

func openRouterEmbed(client *openrouter.OpenRouter, model string) embedFunc {
	return func(ctx context.Context, inputs []string) ([][]float32, error) {
		resp, err := client.Embeddings.Generate(ctx, operations.CreateEmbeddingsRequest{
			Model: model,
			Input: operations.CreateInputUnionArrayOfStr(inputs),
		})
		if err != nil {
			return nil, fmt.Errorf("openrouter: %w", err)
		}
		if resp == nil || resp.CreateEmbeddingsResponseBody == nil {
			return nil, fmt.Errorf("openrouter: empty response")
		}

		data := resp.CreateEmbeddingsResponseBody.Data
		vecs := make([][]float32, len(data))
		for i, d := range data {
			if d.Embedding.ArrayOfNumber == nil {
				return nil, fmt.Errorf("openrouter: unexpected format at index %d", i)
			}
			v := make([]float32, len(d.Embedding.ArrayOfNumber))
			for j, f := range d.Embedding.ArrayOfNumber {
				v[j] = float32(f)
			}
			vecs[i] = v
		}
		return vecs, nil
	}
}

func openAIEmbed(client *openai.Client, model string) embedFunc {
	return func(ctx context.Context, inputs []string) ([][]float32, error) {
		resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Model: openai.EmbeddingModel(model),
			Input: inputs,
		})
		if err != nil {
			return nil, fmt.Errorf("openai: %w", err)
		}

		vecs := make([][]float32, len(resp.Data))
		for i, d := range resp.Data {
			vecs[i] = d.Embedding
		}
		return vecs, nil
	}
}
