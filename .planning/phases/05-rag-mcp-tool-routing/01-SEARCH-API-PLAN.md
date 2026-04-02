---
phase: 5
plan: 1
name: search-api-extension
wave: 1
depends_on: []
requirements: [RAG-02, RAG-03, RAG-04]
files_modified: [internal/rag/store.go, internal/rag/embedder.go]
autonomous: true
---

# Plan 01: Search API Extension

Extending the `Store` and `Embedder` components with semantic search capabilities to support RAG retrieval.

## Tasks

<task id="5-01-01" requirements="RAG-03">
<action>
Modify `internal/rag/store.go` to add `Search` method to the `Store` interface.
Define `SearchResult` struct with fields: `Text`, `Score`, `DocID`, `ChunkIndex`.
Implement `Search(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error)` in `QdrantStore`.
Use Qdrant's `SearchPoints` API with `topK` and basic payload filtering.
Keep only results where `score >= threshold`.
</action>
<read_first>
- internal/rag/store.go
- .planning/phases/03-chunking-embedding-qdrant-storage/03-STORAGE-PLAN.md
</read_first>
<acceptance_criteria>
- `Store` interface updated.
- `QdrantStore.Search` correctly calls Qdrant and filters by score.
- Returns limited results as expected.
</acceptance_criteria>
</task>

<task id="5-01-02" requirements="RAG-02">
<action>
Ensure `internal/rag/embedder.go` provides a way to embed a single string query.
The existing `Embed(ctx context.Context, input interface{}) ([][]float32, error)` should handle `string` input and return `[][]float32` (typically 1 element).
Add a helper method `EmbedQuery(ctx context.Context, query string) ([]float32, error)` for convenience if needed, or simply use the existing one.
</action>
<read_first>
- internal/rag/embedder.go
</read_first>
<acceptance_criteria>
- `Embedder` can be used to generate query vectors.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Search results are filtered by the provided threshold.
- Search results include metadata for citation (DocID, ChunkIndex).
- gRPC query uses the correct collection name and vector size.
</must_haves>

<automated>
- `go test -v ./internal/rag -run TestSearch`
</automated>
