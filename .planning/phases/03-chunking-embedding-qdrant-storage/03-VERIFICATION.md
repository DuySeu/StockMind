---
phase: 03
status: passed
verified: 2026-04-02
---

# Phase 03 Verification: Chunking, Embedding & Qdrant Storage

**Phase Goal:** Text parsed in Phase 2 can be chunked by strategy, embedded via OpenRouter, and stored into Qdrant.

## Must-Haves

| # | Must-Have | Evidence | Status |
|---|-----------|----------|--------|
| 1 | Recursive chunker prioritises `\n\n` over `\n` over ` ` | `splitRecursive` iterates `separators = ["\n\n", "\n", " ", ""]` in order | ✅ |
| 2 | 10% overlap present between consecutive chunks | `mergeWithOverlap` seeds each chunk with `tail = prev[len-overlap:]` | ✅ |
| 3 | All 4 strategies implement `Chunker` interface | `TestStrategyValues` compiles + passes; `go build` clean | ✅ |
| 4 | Embedder Base URL → OpenRouter at `https://openrouter.ai/api/v1` | `cfg.BaseURL = openRouterBaseURL` constant in `embedder.go:22` | ✅ |
| 5 | `Dimensions()` returns 2048 | `TestOpenRouterEmbedder_Dimensions` PASS | ✅ |
| 6 | At most 20 chunks per embedding API call | `batchSize = 20`; `Embed` slices with `[start:end]` | ✅ |
| 7 | UUID → valid Qdrant point ID | `qdrant.NewIDUUID(chunkID.String())` - correct v1.17.1 API | ✅ |
| 8 | Payload has `doc_id`, `chunk_index`, `text`, `strategy` | `store.go` `NewValueMap(...)` with all 4 keys | ✅ |
| 9 | Collection created with 2048 dims + cosine (from Phase 1) | `client.go` `VectorParams{Size: 2048, Distance: Distance_Cosine}` | ✅ |
| 10 | Model mismatch detected on startup | `assertModelPoint` returns error if `stored != configuredModel` | ✅ |

## Test Results

- **27/27 tests pass** across `internal/rag` (all phases combined)
- `go build ./...` — clean, zero warnings
- No regressions: prior phase tests (`parser_test.go`) still pass

## Requirements Coverage

| REQ-ID | Plan | Status |
|--------|------|--------|
| PROC-03 | 01-CHUNKERS task 3-01-02 (RecursiveChunker) | ✅ |
| PROC-04 | 01-CHUNKERS task 3-01-03 (Fixed + Paragraph) | ✅ |
| PROC-05 | 01-CHUNKERS task 3-01-04 (SemanticChunker) | ✅ |
| PROC-06 | 03-STORAGE tasks 3-03-01/02 (QdrantStore + consistency) | ✅ |
| PROC-07 | 02-EMBEDDER tasks 3-02-01/02 (OpenRouterEmbedder + batching) | ✅ |

## Automated Verification

```
go test -v ./internal/rag/... → 27 PASS, 0 FAIL (0.668s)
go build ./...                → exit 0
```

## Notes

- `SemanticChunker` requires a live `Embedder` at runtime. Its sentence tokeniser handles Vietnamese/English punctuation without external NLP libraries.
- The model consistency sentinel point (id=0) must be excluded from RAG searches in Phase 5 using `filter: {must: [{key: "chunk_index", range: {gte: 0}}]}`.
- No `langchaingo` dependency was added — all chunking logic uses stdlib.
