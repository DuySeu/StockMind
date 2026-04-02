---
phase: 4
plan: 1
name: async-worker-pipeline
wave: 1
depends_on: []
requirements: [PROC-08, PROC-09]
files_modified: [internal/rag/worker.go, cmd/main.go]
autonomous: true
---

# Plan 01: Async Worker Pipeline

Implement the background processing pipeline for documents, including the worker pool and job orchestration.

## Tasks

<task id="4-01-01" requirements="PROC-08">
<action>
Create `internal/rag/worker.go` with `Job` struct and `Worker` pool.
`Job` should contain: `ID` (UUID), `Name`, `FileType`, `Strategy`, `TempFile`.
`Worker` should hold a buffered channel of `Job` (cap=10) and a reference to dependencies (DB, Parser, Chunker, Embedder, Store).
</action>
<read_first>
- internal/rag/parser.go
- internal/rag/chunker.go (expected from Phase 3)
- internal/rag/embedder.go (expected from Phase 3)
- internal/rag/store.go (expected from Phase 3)
</read_first>
<acceptance_criteria>
- `internal/rag/worker.go` exists with correct structs.
- `go build ./internal/rag` passes.
</acceptance_criteria>
</task>

<task id="4-01-02" requirements="PROC-08">
<action>
Implement `Start(ctx context.Context)` in `internal/rag/worker.go`.
Use a worker pool of size 2.
Use a `sync.WaitGroup` to track active workers and ensure graceful shutdown by listening to `ctx.Done()`.
</action>
<read_first>
- internal/rag/worker.go
</read_first>
<acceptance_criteria>
- Worker pool starts 2 goroutines.
- Workers drain the job channel.
- Workers exit when the context is cancelled.
</acceptance_criteria>
</task>

<task id="4-01-03" requirements="PROC-09">
<action>
Implement the `process(ctx context.Context, job *Job)` internal method.
Steps:
1. Update document status in DB to `processing` (using `UpdateDocumentStatus` from `sqlc`).
2. Open the temporary file.
3. Call `Parser.Parse(file)`.
4. Call `Chunker.Chunk(text)` based on the job strategy.
5. Call `Embedder.Embed(chunks)` in batches of 20.
6. Call `Store.Upsert(vectors, docID, metadata)`.
7. Update document status in DB to `ready` with `chunk_count`.
8. On any error: Update status to `failed` with the error message.
9. Finalize: Cleanup temp file and close the file handle.
</action>
<read_first>
- internal/database/documents.sql.go
- internal/rag/parser.go
- internal/rag/chunker.go
- internal/rag/embedder.go
- internal/rag/store.go
</read_first>
<acceptance_criteria>
- Comprehensive error handling for each pipeline step.
- Document status updated correctly in PostgreSQL throughout the lifecycle.
- Temporary files are deleted regardless of success or failure.
</acceptance_criteria>
</task>

<task id="4-01-04" requirements="PROC-08">
<action>
Wire the worker into `cmd/main.go`.
Initialize the worker with its dependencies.
Start the worker in a separate goroutine.
Ensure the worker context is cancelled during the server shutdown signal.
</action>
<read_first>
- cmd/main.go
</read_first>
<acceptance_criteria>
- Worker starts on application startup.
- Worker shuts down gracefully when SIGINT/SIGTERM is received.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Graceful shutdown: In-flight jobs are allowed to complete if possible, but no new jobs are accepted after shutdown signal.
- Fail-safe: Always cleanup temporary files even if the worker panics (though panics should ideally be recovered).
- Status consistency: The database `documents` table must always reflect the correct state (pending -> processing -> ready/failed).
</must_haves>

<automated>
- `go test -v ./internal/rag -run TestWorker` (Mock dependencies where possible)
</automated>
