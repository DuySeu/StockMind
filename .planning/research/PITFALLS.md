# Pitfalls Research — StockMind RAG Feature

**Domain:** RAG pipeline in Go + Qdrant + LLM intent routing
**Researched:** 2026-04-01
**Confidence:** HIGH

## Critical Pitfalls

### 1. Embedding Model Mismatch (Phase: Ingestion + Retrieval)

**Problem:** Using a different embedding model at query time vs. index time produces semantically incompatible vectors. Search results become garbage silently — no error, just wrong answers.

**Warning signs:** Low relevance scores across all queries, chatbot ignores retrieved context.

**Prevention:**
- Store the embedding model name in the documents table and/or Qdrant collection metadata
- Assert at startup that the configured model matches collection metadata
- Never change the embedding model without re-indexing all documents

---

### 2. Goroutine Leak in Background Worker (Phase: Worker Implementation)

**Problem:** Background goroutines not wired to server shutdown context → leak on graceful shutdown, potentially corrupts in-flight Qdrant upserts.

**Warning signs:** Server takes too long to shut down, Qdrant data partially written.

**Prevention:**
```go
// Worker MUST accept context from server lifecycle
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return  // Graceful exit
        case job := <-w.jobs:
            w.process(ctx, job)
        }
    }
}
```
- Use `errgroup` from `golang.org/x/sync`
- Wire worker context to the same server shutdown context used by existing chi server

---

### 3. Tool Description Too Vague → Misrouting (Phase: MCP Tool)

**Problem:** If `retrieve_knowledge` tool description is generic, LLM will call it for live market data queries or calculation requests → returns no results → confuses the model.

**Warning signs:** Agent calls RAG tool for "giá cổ phiếu VNM hôm nay" and gets empty results.

**Prevention:**
```
// Be explicit about what retrieve_knowledge IS and IS NOT for:
"Use for: definitions, terminology, concepts, background knowledge from uploaded documents.
Do NOT use for: real-time prices, today's market data, live calculations, current news."
```
- Add negative examples to tool description
- Test routing with a matrix of query types before shipping

---

### 4. PDF Parsing Produces Garbage Text (Phase: Parser)

**Problem:** `pdfcpu` text extraction works well for text-based PDFs but can produce garbled output for:
- Scanned PDFs (images of text → no extractable text)
- PDFs with unusual font encodings
- Tables without proper text flow

**Warning signs:** chunk_count is high but content is gibberish like "ffffffff" or random characters.

**Prevention:**
- Validate extracted text quality: check ratio of printable chars to total chars
- If raw text < 100 meaningful chars, mark document as `failed` with message: "Could not extract text from this PDF. Try converting to a text-based PDF."
- Do NOT attempt OCR (out of scope); surface clear error to user

---

### 5. Memory Spike from Large File Processing (Phase: Worker)

**Problem:** Loading entire 10MB PDF into memory for parsing, then generating chunks, then all embeddings in one batch → potential 100MB+ memory spike per job if not streamed.

**Warning signs:** High memory usage during processing, OOM kills.

**Prevention:**
- Process file in streaming fashion where possible
- Batch embedding calls: max 20 chunks per OpenRouter API call
- Use temporary files, not in-memory buffers, for intermediate storage
- Worker pool size = 2 concurrent jobs (not unbounded)

---

### 6. Qdrant Collection Initialization Race (Phase: Startup)

**Problem:** If server starts and begins processing jobs before Qdrant Docker container is fully healthy, collection creation fails silently or panics.

**Warning signs:** First document always fails to index after fresh docker-compose up.

**Prevention:**
- Add health check to Qdrant in docker-compose: `healthcheck: test: ["CMD", "curl", "-f", "http://localhost:6333/healthz"]`
- Add depends_on with condition: service_healthy for backend service
- Implement retry with exponential backoff for Qdrant client init (max 5 retries, 1s→2s→4s→8s→16s)

---

### 7. Score Threshold Too Low → Noisy Context (Phase: MCP Tool)

**Problem:** Returning all top-K results regardless of similarity score → LLM receives irrelevant chunks → hallucinates or gives generic answers citing wrong documents.

**Warning signs:** Chatbot says "According to the document..." but information is clearly wrong.

**Prevention:**
- Set `score_threshold: 0.70` as minimum (cosine similarity)
- If no chunks pass threshold, return: "No relevant information found in knowledge base"
- Log retrieval scores for debugging; tune threshold based on real queries

---

### 8. sqlc Migration Drift (Phase: Database)

**Problem:** Adding new tables without creating a proper goose migration means the DB schema diverges between environments (dev vs. some future prod).

**Warning signs:** "column does not exist" errors in non-dev environments.

**Prevention:**
- Always create a monotonic goose migration for the new `documents` table
- Run `make migrate` as first step, verify sqlc generates correct code before building
- Add migration to docker-compose startup sequence

---

## Phase Mapping

| Pitfall | Phase Where It Bites | Priority |
|---------|---------------------|---------|
| Embedding model mismatch | Phase 1 (DB + Qdrant setup) | 🔴 Critical |
| Tool description vague | Phase 4 (MCP tool) | 🔴 Critical |
| Goroutine leak | Phase 3 (Worker) | 🔴 Critical |
| PDF garbage text | Phase 2 (Parser) | 🟡 High |
| Memory spike | Phase 3 (Worker) | 🟡 High |
| Qdrant startup race | Phase 1 (Infra) | 🟡 High |
| Score threshold noise | Phase 4 (MCP tool) | 🟡 High |
| sqlc migration drift | Phase 1 (DB) | 🟡 High |

---
*Pitfalls research for: StockMind RAG Feature*
*Researched: 2026-04-01*
