# Architecture Research — StockMind RAG Feature

**Domain:** RAG pipeline integration into existing Go backend
**Researched:** 2026-04-01
**Confidence:** HIGH

## Component Overview

```
┌─────────────────────────────────────────────────┐
│                  React Frontend                  │
│  DocumentUpload | DocumentList | Chat (existing) │
└──────────────┬──────────────────────────────────┘
               │ REST API
┌──────────────▼──────────────────────────────────┐
│              Chi Router (existing)               │
│  POST /api/documents  GET /api/documents  DELETE │
└──────────────┬──────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────┐
│           DocumentService (new)                  │
│  - Upload, list, delete document metadata        │
│  - Enqueue processing job                        │
└──────┬───────────────────────┬───────────────────┘
       │                       │
┌──────▼──────┐    ┌───────────▼────────────────────┐
│  PostgreSQL │    │   ProcessingWorker (new, async)  │
│  documents  │    │   - Parse (PDF/DOCX/MD/TXT)      │
│  table      │    │   - Chunk (strategy-based)       │
│  (metadata) │    │   - Embed (OpenRouter API)       │
└─────────────┘    │   - Upsert to Qdrant             │
                   │   - Update doc status in PG      │
                   └───────────────────────────────── ┘
                               │
                   ┌───────────▼────────────┐
                   │      Qdrant (Docker)    │
                   │   Collection: stockmind │
                   │   Vector: 768-dim cosine│
                   └────────────────────────┘

── Chat Path ──────────────────────────────────────
┌─────────────────────────────────────────────────┐
│  Agent Layer (existing /internal/agent)          │
│   └─ LLM sees tool: retrieve_knowledge           │
│       → calls MCP RAG Tool                       │
└──────────────┬──────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────┐
│  RAG MCP Tool (new, in /internal/mcp)            │
│  - Embed the user query (OpenRouter)             │
│  - Search Qdrant top-K                           │
│  - Return formatted context chunks               │
└─────────────────────────────────────────────────┘
```

## New Packages & Files

```
internal/
├── rag/                          ← NEW package
│   ├── chunker.go                ← Chunking strategies interface + implementations
│   ├── chunker_fixed.go
│   ├── chunker_recursive.go
│   ├── chunker_paragraph.go
│   ├── chunker_semantic.go
│   ├── parser.go                 ← Document parsing (PDF, DOCX, MD, TXT)
│   ├── embedder.go               ← OpenRouter embedding calls
│   ├── store.go                  ← Qdrant operations (init collection, upsert, search)
│   └── worker.go                 ← Background processing pipeline
├── service/
│   └── document.go               ← NEW DocumentService (upload, list, delete)
├── mcp/
│   └── rag_tool.go               ← NEW retrieve_knowledge MCP tool
└── server/
    └── document_handler.go       ← NEW HTTP handlers for document CRUD

internal/database/
└── queries/
    └── documents.sql             ← NEW sqlc queries for documents table

migrations/
└── XXXXXXXX_add_documents.sql    ← NEW goose migration
```

## Data Flow — Upload & Processing

```
1. POST /api/documents (multipart/form-data)
   ├── Validate: file type, size ≤ 10MB
   ├── Save file temporarily (os.CreateTemp)
   ├── INSERT into documents table (status=pending)
   ├── Enqueue job to processing channel (buffered chan, cap=10)
   └── Return {id, status: "pending"} → HTTP 202 Accepted

2. Background Worker (goroutine pool, size=2)
   ├── UPDATE status = "processing"
   ├── Parse document → raw text
   ├── Chunk text (selected strategy + params)
   ├── Batch embed chunks (OpenRouter, batch=20)
   ├── Batch upsert to Qdrant (with doc_id payload)
   ├── UPDATE status = "ready", chunk_count = N
   └── Cleanup temp file

3. On failure:
   └── UPDATE status = "failed", error_message = "..."
```

## Data Flow — Chat RAG Retrieval

```
1. User message → Agent (existing)
2. Agent sends to LLM with tools context (including retrieve_knowledge)
3. LLM intent: "this is a terminology/knowledge question"
   → LLM calls: retrieve_knowledge(query="P/E ratio định nghĩa")
4. RAG MCP Tool:
   ├── Embed query via OpenRouter (same model as indexing)
   ├── Search Qdrant: top_k=5, score_threshold=0.7
   ├── Format results: "Context from documents:\n[chunk1]\n[chunk2]..."
   └── Return formatted string to LLM
5. LLM synthesizes answer using retrieved context
6. Stream response to user (existing WebSocket path)
```

## Database Schema (New)

```sql
-- documents table
CREATE TABLE documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    file_type   TEXT NOT NULL,      -- pdf, docx, md, txt
    size_bytes  BIGINT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending|processing|ready|failed
    chunk_count INTEGER,
    strategy    TEXT NOT NULL,      -- fixed|recursive|paragraph|semantic
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Component Boundaries

| Component | Owns | Does NOT own |
|-----------|------|-------------|
| `DocumentService` | Business logic, validation, status management | File parsing, embedding, Qdrant |
| `rag.Worker` | Parse → chunk → embed → store pipeline | HTTP, DB (except status UPDATE) |
| `rag.Store` | Qdrant collection management, upsert, search | Embedding computation |
| `rag.Embedder` | OpenRouter embedding API calls | Chunking, storage |
| `rag_tool.go` (MCP) | Query embedding + Qdrant search + result formatting | Document management |

## Suggested Build Order

1. **Database layer** — migration + sqlc queries for documents table
2. **`internal/rag/store.go`** — Qdrant client init, collection setup, upsert, search
3. **`internal/rag/parser.go`** — PDF + DOCX + MD + TXT parsing
4. **`internal/rag/chunker.go`** — strategy interface + all 4 implementations
5. **`internal/rag/embedder.go`** — OpenRouter embedding via go-openai
6. **`internal/rag/worker.go`** — async processing pipeline
7. **`internal/service/document.go`** — DocumentService
8. **`internal/server/document_handler.go`** — HTTP endpoints
9. **`internal/mcp/rag_tool.go`** — MCP tool registration
10. **Frontend** — Upload UI + Document management page

---
*Architecture research for: StockMind RAG Feature*
*Researched: 2026-04-01*
