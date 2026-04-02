# Phase 3: Chunking, Embedding & Qdrant Storage - Context

**Gathered:** 2026-04-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Taking extracted text from Phase 2 and transforming it into vector embeddings stored in Qdrant. This requires implementing multiple chunking strategies (recursive, fixed, paragraph, semantic), calling the OpenRouter embedding API, and managing the Qdrant storage lifecycle.

</domain>

<decisions>
## Implementation Decisions

### Chunking Strategies
- **D-01:** **Recursive Splitting** (Default) — Target 512 tokens with 10% (approx 51) overlap. High-quality context preservation for most doc types.
- **D-02:** **Fixed-size** — Exact token counts for predictable memory/storage impact.
- **D-03:** **Paragraph-based** — Splits strictly at double-newline or structural boundaries (`\n\n`), ideal for MD/DOCX with well-defined sections.
- **D-04:** **Semantic** — Uses cosine similarity between adjacent sentence embeddings to find "break points" where the topic shifts.
- **D-05:** **Strategy Selection** — Must be selectable per-document during upload (Phase 4 integration) via internal API parameters.

### Embedding Pipeline
- **D-06:** **OpenRouter Integration** — Use `nvidia/llama-nemotron-embed-vl-1b-v2:free` (2048-dim) for cost-free, high-performance embeddings.
- **D-07:** **Batch Size** — Use `batch=20` per API call to minimize RTT while staying within typical rate limits for free models.
- **D-08:** **Retry Policy** — 3 retries with exponential backoff for 429/5xx errors on OpenRouter side.

### Qdrant Storage
- **D-09:** **Payload Schema** — Store `{doc_id: uuid, chunk_index: int, text: string, strategy: string}`.
- **D-10:** **Consistency Assert** — Store `embedding_model` name in Qdrant collection params; error during startup if system configured model mismatch with existing collection.

### the agent's Discretion
- Chunk boundary characters selection (which delimiters to prioritize).
- Exact backoff duration for API retries.
- Specific sentence-splitting library selection for recursive/semantic chunking.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core Backend & Infrastructure
- `internal/rag/parser.go` — Input source for extracted text from Phase 2.
- `internal/database/queries.sql` — SQLC definitions for document metadata tracking.

### Vector Storage
- `https://qdrant.tech/documentation/concepts/points/` — Qdrant point/payload management.
- `github.com/qdrant/go-client` — Official gRPC Go client.

### Embedding API
- `https://openrouter.ai/docs#embeddings` — OpenRouter embedding specification.
- `https://github.com/sashabaranov/go-openai` — SDK used for OpenRouter communication.

</canonical_refs>
