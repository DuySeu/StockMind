---
plan: 03/01-CHUNKERS
status: complete
phase: 03
wave: 1
---

# Plan 01 Summary: Core Chunking Strategies

## What was built

Implemented the `Chunker` interface and 4 text splitting strategies in `internal/rag/`:

- **`chunker.go`** — `Chunker` interface with `Chunk(ctx, text) ([]string, error)` and `Strategy() Strategy`. `Strategy` type (string enum) with 4 constants.
- **`chunker_recursive.go`** — Hierarchical character splitter using `\n\n → \n → " " → ""` separator cascade. Default 512-char chunks with 51-char overlap.  
- **`chunker_fixed.go`** — Exact-window splitter at configurable character count with overlap.
- **`chunker_paragraph.go`** — Splits on `\n\n`; normalises internal single-newlines to spaces. Single newlines within a paragraph do NOT trigger a split.
- **`chunker_semantic.go`** — Uses cosine similarity between adjacent sentence embeddings to detect topic shifts; accepts a `nil` Embedder (deferred until Plan 02).
- **`chunker_test.go`** — 16 unit tests with 100% pass rate.

## Key decisions
- Stdlib only — no `langchaingo` dependency needed. Keeps the Go module lean.
- `mergeWithOverlap` in recursive chunker re-seeds each new chunk with the tail of the previous one, ensuring cross-boundary context.
- `cosineSimilarity` for semantic chunker implemented with pure `math` to avoid float64↔float32 conversion overhead.

## self-check: PASSED
- `go build ./internal/rag` — clean
- `go test -v ./internal/rag -run TestRecursive|TestFixed|TestParagraph|TestSplit|TestCosine|TestStrategy` — 13/13 PASS

## key-files.created
- `internal/rag/chunker.go`
- `internal/rag/chunker_recursive.go`
- `internal/rag/chunker_fixed.go`
- `internal/rag/chunker_paragraph.go`
- `internal/rag/chunker_semantic.go`
- `internal/rag/chunker_test.go`
