---
phase: 3
plan: 1
name: core-chunking-strategies
wave: 1
depends_on: []
requirements: [PROC-03, PROC-04, PROC-05]
files_modified: [internal/rag/chunker.go, internal/rag/chunker_fixed.go, internal/rag/chunker_recursive.go, internal/rag/chunker_paragraph.go, internal/rag/chunker_semantic.go]
autonomous: true
---

# Plan 01: Core Chunking Strategies

Implement the `Chunker` interface and multiple splitting strategies to prepare text for embedding.

## Tasks

<task id="3-01-01" requirements="PROC-03">
<action>
Create `internal/rag/chunker.go` with `Chunker` interface and `Strategy` enum.
Include `Recursive`, `Fixed`, `Paragraph`, and `Semantic` constants.
`Chunker` interface should have `Chunk(ctx context.Context, text string) ([]string, error)` method.
</action>
<read_first>
- internal/rag/parser.go (to see the context of input text)
</read_first>
<acceptance_criteria>
- `internal/rag/chunker.go` exists and defines the interface and enum.
- `go build ./internal/rag` passes.
</acceptance_criteria>
</task>

<task id="3-01-02" requirements="PROC-04">
<action>
Implement `internal/rag/chunker_recursive.go` using `github.com/tmc/langchaingo/textsplitter`.
Set default target 512 characters/tokens with 10% overlap (~51 chars/tokens).
Use hierarchal separators: `\n\n`, `\n`, `" "`, `""`.
</action>
<read_first>
- internal/rag/chunker.go
</read_first>
<acceptance_criteria>
- `internal/rag/chunker_recursive.go` implements the `Chunker` interface.
- Sample passage split into ~512 unit chunks with overlap.
</acceptance_criteria>
</task>

<task id="3-01-03" requirements="PROC-04">
<action>
Implement `internal/rag/chunker_fixed.go` and `internal/rag/chunker_paragraph.go`.
`Fixed` strategy splits exactly at a character/token count.
`Paragraph` strategy splits strictly at `\n\n`.
</action>
<read_first>
- internal/rag/chunker.go
</read_first>
<acceptance_criteria>
- Both implementations exist and follow the interface.
- `ParagraphChunker` handles single-newline cases correctly (treats continuous text as one chunk until double-newline).
</acceptance_criteria>
</task>

<task id="3-01-04" requirements="PROC-05">
<action>
Implement `internal/rag/chunker_semantic.go` skeleton.
Since this requires embeddings, define it to take an `Embedder` (to be implemented in Plan 2) in its constructor.
Implementation:
1. Split into sentences.
2. Embed each sentence.
3. Calculate cosine similarity between neighbors.
4. Split where similarity < threshold (default 0.70).
</action>
<read_first>
- .planning/phases/03-chunking-embedding-qdrant-storage/03-RESEARCH.md
</read_first>
<acceptance_criteria>
- `SemanticChunker` exists.
- Basic units (sentences) are parsed correctly.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Recursive chunker preserves context by prioritizing structural splits (\n\n).
- 10% overlap is present between consecutive chunks.
- All strategies implement the same Go interface.
</must_haves>

<automated>
- `go test -v ./internal/rag -run TestChunker`
</automated>
