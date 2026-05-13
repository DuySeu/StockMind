package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openrouter "github.com/OpenRouterTeam/go-sdk"

	llm "stockmind/internal/llm"
)

const (
	// EmbeddingModel is the free multimodal embedding model on OpenRouter.
	EmbeddingModel = "nvidia/llama-nemotron-embed-vl-1b-v2:free"

	// embeddingDimensions is the vector size returned by EmbeddingModel.
	embeddingDimensions = 2048

	// defaultBatchSize is the number of text chunks sent per embedding API call.
	defaultBatchSize = 20
)

// Embedder converts a batch of strings into float32 embedding vectors.
// Implementations must be safe for concurrent use.
type Embedder interface {
	Embed(ctx context.Context, input []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
	Dimensions() int
}

// OpenRouterEmbedder uses the shared OpenRouter SDK client for embeddings.
type OpenRouterEmbedder struct {
	client    *openrouter.OpenRouter
	batchSize int
}

// NewOpenRouterEmbedder constructs an OpenRouterEmbedder using the shared client.
// Use batchSize <= 0 to use the defaultBatchSize (20).
func NewOpenRouterEmbedder(client *openrouter.OpenRouter, batchSize int) *OpenRouterEmbedder {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	return &OpenRouterEmbedder{client: client, batchSize: batchSize}
}

// Dimensions returns 2048, the vector size of EmbeddingModel.
func (e *OpenRouterEmbedder) Dimensions() int { return embeddingDimensions }

// Embed generates embeddings for all inputs, batching at most batchSize items per API call.
func (e *OpenRouterEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("embedder: at least one input is required")
	}

	results := make([][]float32, len(inputs))
	for start := 0; start < len(inputs); start += e.batchSize {
		end := start + e.batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		vecs, err := llm.OpenRouterEmbedding(ctx, e.client, EmbeddingModel, inputs[start:end])
		if err != nil {
			return nil, fmt.Errorf("embedder: batch [%d:%d] failed: %w", start, end, err)
		}
		copy(results[start:end], vecs)
	}
	return results, nil
}

// EmbedQuery generates an embedding for a single string.
func (e *OpenRouterEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("embedder: query must not be empty")
	}
	res, err := e.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("embedder: no embedding returned for query")
	}
	return res[0], nil
}
