---
phase: 4
plan: 2
name: document-service
wave: 2
depends_on: [async-worker-pipeline]
requirements: [UPLOAD-01, UPLOAD-03, UPLOAD-04, UPLOAD-05, DOC-01, DOC-02, DOC-03]
files_modified: [internal/service/document.go]
autonomous: true
---

# Plan 02: Document Service

Implement the `DocumentService` to manage document metadata, file uploads, and processing task enqueuing.

## Tasks

<task id="4-02-01" requirements="UPLOAD-01, UPLOAD-03">
<action>
Create `internal/service/document.go` with `DocumentService` struct.
The service should have access to: `*database.Queries`, `*rag.Worker`.
Implement `Upload(ctx context.Context, name, fileType string, size int64, file io.Reader, strategy rag.Strategy) (*database.Document, error)`.
Steps:
1. Save the file to a temporary location using `os.CreateTemp`.
2. Insert a document record in PostgreSQL with status `pending`.
3. Create a `rag.Job` and enqueue it using `Worker.Enqueue(job)`.
4. Return the database record to the caller.
</action>
<read_first>
- internal/database/documents.sql.go
- internal/rag/worker.go
</read_first>
<acceptance_criteria>
- `DocumentService.Upload` saves the file, inserts metadata into the DB, and enqueues the job.
- Proper cleanup (deleting temp file) if enqueuing fails.
</acceptance_criteria>
</task>

<task id="4-02-02" requirements="DOC-01, DOC-02">
<action>
Implement `List(ctx context.Context)` and `GetByID(ctx context.Context, id uuid.UUID)`.
These methods should wrap the corresponding `sqlc` queries.
</action>
<read_first>
- internal/database/documents.sql.go
</read_first>
<acceptance_criteria>
- `List` returns all documents ordered by creation date.
- `GetByID` returns a single document or an appropriate error if not found.
</acceptance_criteria>
</task>

<task id="4-02-03" requirements="DOC-03">
<action>
Implement `Delete(ctx context.Context, id uuid.UUID)` in `DocumentService`.
Steps:
1. Delete the record from PostgreSQL.
2. Call `Store.DeleteVectors(docID)` to remove associated chunks from Qdrant.
</action>
<read_first>
- internal/database/documents.sql.go
- internal/rag/store.go
</read_first>
<acceptance_criteria>
- Document is removed from both PostgreSQL and Qdrant.
- Graceful handling if the document record already doesn't exist? (ensure it's idempotent).
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Document records are correctly initialized with the `pending` status.
- Qdrant vectors are always deleted when a document is removed from the system.
- Interaction between the service and the worker is via a thread-safe channel.
</must_haves>

<automated>
- `go test -v ./internal/service -run TestDocumentService`
</automated>
