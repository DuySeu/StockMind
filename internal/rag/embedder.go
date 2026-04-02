package rag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sethvargo/go-retry"
)

const (
	// openRouterBaseURL is the OpenRouter API base for OpenAI-compatible endpoints.
	openRouterBaseURL = "https://openrouter.ai/api/v1"

	// embeddingModel is the free multimodal embedding model on OpenRouter.
	// Confirmed $0 cost as of Apr 2026. Vector dimensionality: 2048.
	embeddingModel = "nvidia/llama-nemotron-embed-vl-1b-v2:free"

	// embeddingDimensions is the vector size returned by embeddingModel.
	embeddingDimensions = 2048

	// defaultBatchSize is the number of text chunks sent per embedding API call.
	// Chosen to balance network round-trips with free-tier rate limits.
	defaultBatchSize = 20

	// maxRetries is the number of retry attempts for transient API errors.
	maxRetries = 3
)

// Embedder converts a batch of strings (or any input) into float32 embedding
// vectors.  Implementations must be safe for concurrent use.
type Embedder interface {
	// Embed generates embeddings for a slice of strings. The returned slice
	// has the same length as the input; Embed[i] is the embedding for input[i].
	Embed(ctx context.Context, input []string) ([][]float32, error)

	// Dimensions returns the vector length produced by this embedder.
	Dimensions() int
}

// OpenRouterEmbedder calls the OpenRouter /v1/embeddings endpoint using the
// go-openai SDK with a base-URL override. It batches requests and retries
// transient HTTP errors with exponential backoff.
type OpenRouterEmbedder struct {
	client    *openai.Client
	batchSize int
}

// NewOpenRouterEmbedder constructs an OpenRouterEmbedder using the provided
// API key. Use batchSize <= 0 to use the defaultBatchSize (20).
func NewOpenRouterEmbedder(apiKey string, batchSize int) (*OpenRouterEmbedder, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("embedder: OPENROUTER_API_KEY must not be empty")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = openRouterBaseURL
	// Use a custom HTTP client with a reasonable timeout so the embedder does
	// not block indefinitely on slow responses.
	cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}

	return &OpenRouterEmbedder{
		client:    openai.NewClientWithConfig(cfg),
		batchSize: batchSize,
	}, nil
}

// Dimensions returns 2048, the vector size of embeddingModel.
func (e *OpenRouterEmbedder) Dimensions() int { return embeddingDimensions }

// Embed generates embeddings for all inputs, batching at most batchSize items
// per API call. Transient failures (429 / 5xx) are retried with fibonacci
// backoff (up to maxRetries attempts per batch).
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
		batch := inputs[start:end]

		vecs, err := e.embedBatchWithRetry(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embedder: batch [%d:%d] failed: %w", start, end, err)
		}
		copy(results[start:end], vecs)
	}
	return results, nil
}

// embedBatchWithRetry calls the OpenRouter embedding endpoint for a single
// batch, retrying on transient errors.
func (e *OpenRouterEmbedder) embedBatchWithRetry(ctx context.Context, batch []string) ([][]float32, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(maxRetries, b)

	var vecs [][]float32
	err := retry.Do(ctx, b, func(ctx context.Context) error {
		resp, apiErr := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
			Input: batch,
			Model: openai.EmbeddingModel(embeddingModel),
		})
		if apiErr != nil {
			// Treat rate-limit and server errors as retryable; others are fatal.
			if isRetryable(apiErr) {
				return retry.RetryableError(apiErr)
			}
			return apiErr
		}

		vecs = make([][]float32, len(resp.Data))
		for i, d := range resp.Data {
			v := make([]float32, len(d.Embedding))
			for j, f := range d.Embedding {
				v[j] = float32(f)
			}
			vecs[i] = v
		}
		return nil
	})
	return vecs, err
}

// isRetryable returns true for HTTP 429 (Too Many Requests) and 5xx class
// errors from the OpenRouter API.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// go-openai wraps HTTP status codes in the error message.
	retrySignals := []string{"429", "500", "502", "503", "504", "rate limit"}
	for _, sig := range retrySignals {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(sig)) {
			return true
		}
	}
	return false
}
