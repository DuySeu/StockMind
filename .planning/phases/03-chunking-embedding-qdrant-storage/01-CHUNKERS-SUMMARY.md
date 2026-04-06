---
plan: 03/01-CHUNKERS
status: complete
phase: 03
wave: 1
---

# Plan 01 Summary: Core Chunking Strategies

## What was built

Implemented the **`Chunker`** interface and 4 text splitting strategies in a single consolidated file **`internal/rag/chunker.go`**:

- **`chunker.go`** — `Chunker` interface with `Chunk(ctx, text) ([]string, error)` and `Strategy() Strategy`. `Strategy` type (string enum) with 4 constants.
- **Recursive Strategy** — Hierarchical character splitter using `\n\n → \n → " " → ""` separator cascade. Default 512-char chunks with 51-char overlap.  
- **Fixed Strategy** — Exact-window splitter at configurable character count with overlap.
- **Paragraph Strategy** — Splits on `\n\n`; normalises internal single-newlines to spaces. Single newlines within a paragraph do NOT trigger a split.
- **Semantic Strategy** — Uses cosine similarity between adjacent sentence embeddings to detect topic shifts.
- **`chunker_test.go`** — 16 unit tests with 100% pass rate.

## Key decisions
- **Consolidation**: Following Effective Go principles, all related chunking logic is kept in a single file (~420 lines) for better discoverability and cleaner package structure.
- **Selective splitting**: `mergeWithOverlap` in recursive chunker re-seeds each new chunk with the tail of the previous one, ensuring cross-boundary context.
- **Internal helpers**: `splitSentences`, `splitRecursive`, `mergeWithOverlap`, and `cosineSimilarity` are kept as unexported helpers within `chunker.go`.

## self-check: PASSED
- `go build ./internal/rag` — clean
- `go test -v ./internal/rag` — 100% PASS

## key-files.created
- `internal/rag/chunker.go`
- `internal/rag/chunker_test.go`
