# RAG Module Refactor: `internal/knowledge_base`

## Goal

Extract RAG logic from `internal/llm/rag/` and `internal/qdrant/` into a unified `internal/knowledge_base/` package that supports multiple search strategies (semantic, keyword, hybrid) and is easy to extend with new strategies without modifying existing code.

## Current State

RAG is scattered across:
- `internal/llm/rag/` — parser, chunker, embedder
- `internal/qdrant/` — Qdrant gRPC client (Store)
- `internal/mcp/rag_tool.go` — retrieval handler (manual init, embed + search)
- `internal/service/worker/` — ingestion pipeline

Search is semantic-only (cosine similarity on dense vectors).

## New Structure

```
internal/knowledge_base/
├── knowledge_base.go       # Core interfaces, types, factory function
├── store.go                # QdrantStore (moved from internal/qdrant/)
├── ingest.go               # IngestPipeline: parse → chunk → embed → BM25 → upsert
├── parser.go               # Parser interface + PDF/DOCX/MD/TXT (moved from llm/rag/)
├── chunker.go              # Chunker interface + 4 strategies (moved from llm/rag/)
├── embedder.go             # Embedder interface + OpenRouterEmbedder (moved from llm/rag/)
├── bm25.go                 # BM25 tokenizer → sparse vector generation
├── search_semantic.go      # SemanticStrategy: embed query → dense search
├── search_keyword.go       # KeywordStrategy: BM25 → sparse search
├── search_hybrid.go        # HybridStrategy: dense + sparse → RRF fusion
```

Packages removed after migration:
- `internal/qdrant/`
- `internal/llm/rag/`

## Core Interfaces

```go
package knowledge_base

type SearchMode string

const (
    SearchSemantic SearchMode = "semantic"
    SearchKeyword  SearchMode = "keyword"
    SearchHybrid   SearchMode = "hybrid"
)

type SearchResult struct {
    Text       string
    Score      float32
    DocID      string
    ChunkIndex int
}

type SparseVector struct {
    Indices []uint32
    Values  []float32
}

type Retriever interface {
    Search(ctx context.Context, query string, mode SearchMode, topK int) ([]SearchResult, error)
}

type Store interface {
    Upsert(ctx context.Context, docID string, chunks []string, dense [][]float32, sparse []SparseVector) error
    Delete(ctx context.Context, docID string) error
    SearchDense(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error)
    SearchSparse(ctx context.Context, vector SparseVector, topK int) ([]SearchResult, error)
    SearchHybrid(ctx context.Context, dense []float32, sparse SparseVector, topK int) ([]SearchResult, error)
}

type Embedder interface {
    Embed(ctx context.Context, input []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimensions() int
}
```

## Factory

```go
type Config struct {
    QdrantHost     string
    QdrantPort     int
    OpenRouterKey  string
    DBPool         *pgxpool.Pool
}

type KnowledgeBase struct {
    Retriever Retriever
    Pipeline  *IngestPipeline
    Store     Store
}

func New(ctx context.Context, cfg Config) (*KnowledgeBase, error)
```

## Qdrant Collection Schema

```
Collection: "stockmind_knowledge"
Named Vectors:
  - "dense":  size=2048, distance=Cosine
  - "sparse": sparse vector (variable size)
Payload:
  - doc_id: string
  - chunk_index: int
  - text: string
  - strategy: string
```

## Data Flow

### Ingestion

```
POST /v1/documents/
  → worker.Enqueue(job)
  → IngestPipeline.Process(docID, file, fileType, strategy):
      1. parser.Parse(file)         → raw text
      2. chunker.Chunk(text)        → []string
      3. embedder.Embed(chunks)     → [][]float32 (dense)
      4. bm25.Vectorize(chunks)     → []SparseVector (sparse)
      5. store.Upsert(docID, chunks, dense, sparse)
```

### Retrieval

```
retrieve_knowledge(query, mode="hybrid")
  → Retriever.Search(query, "hybrid", topK=5):
      1. embedder.EmbedQuery(query) → dense vector
      2. bm25.VectorizeQuery(query) → sparse vector
      3. store.SearchHybrid(dense, sparse, topK)
         → Qdrant QueryPoints: prefetch dense + prefetch sparse → RRF fusion
      4. return []SearchResult
```

## BM25 Implementation

- Hash-based term indexing: `crc32(term) % 30000` — no fixed vocabulary file needed
- IDF: `log(N / df(term))` where N = total chunks, df = document frequency
- Tokenization: whitespace + punctuation split, lowercase, Vietnamese + English stopwords removal
- IDF persistence: PostgreSQL table `knowledge_base_config` (1 row, JSONB)
- Rebuild trigger: when document count changes >20% or manual

## Integration Changes

| File | Before | After |
|------|--------|-------|
| `mcp/rag_tool.go` | Manual init, embed + search | Inject `Retriever`, call `.Search()` |
| `service/worker/worker.go` | Import qdrant + llm/rag | Inject `IngestPipeline`, call `.Process()` |
| `service/service.go` | Create embedder + worker | Create `knowledge_base.New()` |
| `server/document.handler.go` | Import `llm/rag` for Strategy | Import `knowledge_base` |
| `server/server.go` | Hold `qdrant.Store` | Hold `knowledge_base.Store` |
| `cmd/main.go` | Init Qdrant separately | Init `knowledge_base.New(cfg)` |

## Error Handling

```go
var (
    ErrEmptyQuery   = errors.New("query must not be empty")
    ErrNoResults    = errors.New("no relevant results found")
    ErrStoreUnavail = errors.New("vector store unavailable")
    ErrEmbedFailed  = errors.New("embedding generation failed")
)
```

Graceful degradation: if sparse search fails but dense succeeds → fallback to semantic-only, log warning.

## Testing

- **BM25 unit tests:** tokenize, IDF, sparse vector output (pure logic)
- **Parser/chunker tests:** move existing tests from `llm/rag`
- **Store integration tests:** testcontainers or mock gRPC
- **Interface mocking:** all core types are interfaces → easy to mock in consumer tests

## Migration Notes

- Collection schema change (unnamed vector → named vectors) requires re-indexing existing documents
- Migration strategy TBD as separate implementation task
- Ensure backward compatibility: if collection has old schema, fallback to semantic-only until re-indexed
