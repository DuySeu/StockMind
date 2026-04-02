---
phase: 3
plan: 2
name: embedder-integration
wave: 1
depends_on: []
requirements: [PROC-07]
files_modified: [internal/rag/embedder.go]
autonomous: true
---

# Plan 02: OpenRouter Embedder Integration

Implement the `Embedder` component to connect the system to OpenRouter's embedding API.

## Tasks

<task id="3-02-01" requirements="PROC-07">
<action>
Create `internal/rag/embedder.go` with `Embedder` interface.
`Embedder` should have `Embed(ctx context.Context, input interface{}) ([][]float32, error)` and `Dimensions() int`.
Add an implementation `OpenRouterEmbedder` using `github.com/sashabaranov/go-openai`.
Override base URL to `https://openrouter.ai/api/v1` and use model `nvidia/llama-nemotron-embed-vl-1b-v2:free`.
Fix dimensions to 2048.
</action>
<read_first>
- internal/rag/parser_test.go (for env var loading patterns)
- .planning/phases/03-chunking-embedding-qdrant-storage/03-RESEARCH.md
</read_first>
<acceptance_criteria>
- `internal/rag/embedder.go` correctly initializes with `OPENROUTER_API_KEY`.
- `Embedder` interface includes `Dimensions()` returning 2048.
- `go test ./internal/rag` (mocked) passes.
</acceptance_criteria>
</task>

<task id="3-02-02" requirements="PROC-07">
<action>
Implement batching and retry logic in `OpenRouterEmbedder`.
Batch size: 20 chunks per request.
Retry: `github.com/avast/retry-go` or manual exponential backoff (3 attempts).
Handle `429` (Rate Limit) and `5xx` (Server Errors).
</action>
<read_first>
- internal/rag/embedder.go
</read_first>
<acceptance_criteria>
- `Embedder` handles a slice of strings (>20) by automatically batching into 20-unit requests.
- Exponential backoff is verifiable in code.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- API Base URL points correctly to OpenRouter.
- Returns float32 vector with 2048 dimensions.
- Correctly handles batch requests with at most 20 elements per API call.
</must_haves>

<automated>
- `go test -v ./internal/rag -run TestEmbedderBatching`
</automated>
