# Research Summary — StockMind RAG Feature

**Synthesized:** 2026-04-01
**Research Files:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md

---

## Key Findings

### 1. Stack is Simpler Than Expected

The existing StockMind stack handles RAG well with minimal additions:
- **go-openai SDK already in codebase** — just override base URL to OpenRouter for embeddings. Zero new SDK needed.
- **MCP framework already in place** — register `retrieve_knowledge` as another MCP tool; the routing infrastructure is free.
- **Only new external dependency:** `github.com/qdrant/go-client` + `github.com/pdfcpu/pdfcpu`
- **DOCX parsing** uses stdlib (`archive/zip` + `encoding/xml`) — no extra dependency.
- **Confirmed embedding model:** `nvidia/llama-nemotron-embed-vl-1b-v2:free` — the **only $0 free model** on OpenRouter (confirmed via API Apr 2026), 2048-dim, 131K context, multimodal-capable

### 2. Architecture Fits Cleanly Into Existing Layers

New `internal/rag/` package handles all RAG-specific logic. No existing packages need structural changes:
- `DocumentService` → new service, plugs into existing service layer pattern
- `rag_tool.go` → new MCP tool, registers alongside existing Tavily/DB tools  
- Document HTTP handlers → new, follow existing handler patterns in `/internal/server`

The async processing worker uses a **buffered channel + goroutine pool** wired to the server's shutdown context — fits cleanly into the existing server lifecycle.

### 3. Chunking Strategy — Simplified Recommendation

Despite 4+ strategies in literature, practical recommendation for this use case:

| Strategy | Label | Recommended For |
|----------|-------|----------------|
| `recursive` | Smart Split ⭐ | **Default for most docs** |
| `fixed` | Fixed Size | Uniform structured reports |
| `paragraph` | By Paragraph | Articles, research papers |
| `semantic` | By Topic | Mixed-topic documents |

**Default params:** 512 tokens, 10% overlap. Let user pick strategy at upload time.

> Note: With `nvidia/llama-nemotron-embed-vl-1b-v2:free` having 131K context, chunk size limit is effectively irrelevant. Keep 512 tokens for retrieval precision, not due to model constraint.

### 4. Three Critical Non-Negotiables

From PITFALLS.md, these must be built correctly from day one:

**① Embedding model consistency** — Store `nvidia/llama-nemotron-embed-vl-1b-v2:free` + dimension `2048` in DB and Qdrant collection metadata; assert at startup they match.

**② RAG tool description precision** — Tool description must explicitly state what it IS and ISN'T for (live prices vs. knowledge base). This is the entire routing mechanism.

**③ Worker context wiring** — Background goroutines must respect server shutdown context to prevent leaks and partial writes.

### 5. UX Flow Is Clear

```
Upload: select file → validates in browser → POST → 202 Accepted → 
  status badge "Processing" → poll every 3s → "Ready" (or "Failed" with reason)

Chat: user asks question → LLM decides → calls retrieve_knowledge tool →
  Qdrant similarity search (cosine ≥ 0.70) → returns top-5 chunks →
  LLM synthesizes answer
```

---

## Recommended Phase Structure

Based on build order dependencies:

| Phase | Focus | Key Deliverables |
|-------|-------|-----------------|
| **Phase 1** | Infrastructure + DB | Qdrant Docker, documents migration, sqlc queries, Qdrant client init with retry |
| **Phase 2** | Document Parsing | PDF (pdfcpu), DOCX (stdlib), MD, TXT parsers; text quality validation |
| **Phase 3** | Chunking + Embedding | 4 chunking strategies, OpenRouter embedder, batch upsert to Qdrant |
| **Phase 4** | Async Worker + Service | Worker goroutine pool, DocumentService, HTTP endpoints, status tracking |
| **Phase 5** | MCP Tool + Routing | `retrieve_knowledge` tool with precise description, score threshold |
| **Phase 6** | Frontend UI | Upload form with strategy picker, document list, status indicators |

---

## Open Questions for Roadmap

1. **Qdrant collection name:** Use `stockmind_knowledge` as the single global collection name.
2. **Worker pool size:** 2 concurrent jobs is safe for 10MB max, 50 doc limit.
3. **Score threshold:** Start at `0.70` — can be tuned via env var later.
4. **Top-K:** Return 5 chunks by default — good balance of context vs. token usage.
5. **Temp file cleanup:** Always delete temp file after processing (success or failure).

---
*Research synthesis for: StockMind RAG Feature*
*Synthesized: 2026-04-01*
