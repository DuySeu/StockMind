# Phase 4: Async Worker & Document Service + REST API - Context

**Gathered:** 2026-04-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Establishing the orchestration and API layer for document management. This phase transforms the isolated RAG components (parser, chunker, embedder, store) into a functional background processing pipeline. It includes handling file uploads via REST, managing document metadata in PostgreSQL, and enqueuing processing jobs to an asynchronous worker pool.

</domain>

<decisions>
## Implementation Decisions

### Async Worker Pipeline
- **D-01:** **Goroutine Pool** — Use a worker group of size 2 to prevent CPU/Memory exhaustion during large file processing.
- **D-02:** **Buffered Channel** — Use a `chan Job` with `capacity=10`. This provides backpressure to the API layer.
- **D-03:** **Status Lifecycle** — `pending` (uploaded) -> `processing` (worker started) -> `ready` (vector store indexed) OR `failed` (error occurred).
- **D-04:** **Graceful Shutdown** — Workers must respect context cancellation and finish in-flight jobs before the process exits.

### Document Service
- **D-05:** **File Storage** — Use `os.CreateTemp` for temporary storage during processing. No permanent local file storage is required since vectors are the source of truth.
- **D-06:** **Cleanup** — Implementation MUST ensure temporary files are deleted in all scenarios (success, failure, panic).
- **D-07:** **UUIDs** — All documents are identified by a unique Google/UUID.

### REST API
- **D-08:** **Endpoints** — `POST /api/documents`, `GET /api/documents`, `GET /api/documents/:id`, `DELETE /api/documents/:id`.
- **D-09:** **Multipart Limit** — Limit uploads to 10MB per file via `http.MaxBytesReader`.
- **D-10:** **Response codes** — 202 Accepted for uploads (async processing), 200/204 for other operations.

### the agent's Discretion
- Context timeout value for internal DB/Qdrant calls (recommended: 10-30s).
- Location of temporary files (system default vs. repo `/tmp`). Use system default unless specified.
- Specific JSON error message format (must align with existing API error response style).

</decisions>

<canonical_refs>
## Canonical References

### Core Backend
- `internal/server/routes.go` — Registration point for new HTTP handlers.
- `internal/database/queries/documents.sql` — SQLC queries for document status tracking.

### RAG Components (Phase 2 & 3)
- `internal/rag/parser.go` — Document text extraction.
- `internal/rag/chunker.go` — Text splitting strategies.
- `internal/rag/embedder.go` — Vector embedding generation.
- `internal/rag/store.go` — Vector database operations.

### Go Concurrency
- `https://go.dev/doc/effective_go#concurrency` — Best practices for goroutines and channels.
- `https://pkg.go.dev/golang.org/x/sync/errgroup` — Recommended for worker pool management.

</canonical_refs>
