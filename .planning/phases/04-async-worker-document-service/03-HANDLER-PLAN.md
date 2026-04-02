---
phase: 4
plan: 3
name: rest-api-handlers
wave: 3
depends_on: [document-service]
requirements: [UPLOAD-01, UPLOAD-04, DOC-01, DOC-02, DOC-03]
files_modified: [internal/server/document_handler.go, internal/server/routes.go, internal/server/server.go]
autonomous: true
---

# Plan 03: REST API & Handlers

Implement the RESTful endpoints for document management and integrate them into the server routing.

## Tasks

<task id="4-03-01" requirements="UPLOAD-01, UPLOAD-04">
<action>
Create `internal/server/document_handler.go`.
Implement `UploadDocumentHandler(w http.ResponseWriter, r *http.Request)`.
Steps:
1. Limit the request body size to 10MB using `http.MaxBytesReader`.
2. Parse the multipart form.
3. Extract `name`, `strategy` (defaulting to `recursive` if empty), and the file.
4. Call `DocumentService.Upload`.
5. Return 202 Accepted with the document metadata.
</action>
<read_first>
- internal/service/document.go
- internal/server/routes.go (for multipart parsing examples)
</read_first>
<acceptance_criteria>
- Handler handles multipart uploads.
- 413 "Request Entity Too Large" returned for files > 10MB.
- 202 "Accepted" returned on successful enqueue.
</acceptance_criteria>
</task>

<task id="4-03-02" requirements="DOC-01, DOC-02, DOC-03">
<action>
Implement `ListDocumentsHandler`, `GetDocumentHandler`, and `DeleteDocumentHandler` in `internal/server/document_handler.go`.
Use standard JSON responses.
Ensure `DeleteDocumentHandler` handles the UUID parameter from the URL.
</action>
<read_first>
- internal/service/document.go
</read_first>
<acceptance_criteria>
- Handlers exist and delegate work to `DocumentService`.
- 404 "Not Found" returned for non-existent IDs.
</acceptance_criteria>
</task>

<task id="4-03-03" requirements="UPLOAD-01">
<action>
Update `internal/server/server.go` and `internal/server/routes.go`.
1. Add `DocumentService` field to the `Server` struct.
2. Initialize `DocumentService` in `NewServer`.
3. Register the new routes in `RegisterRoutes`:
   - `POST /api/documents` -> `s.UploadDocumentHandler`
   - `GET /api/documents` -> `s.ListDocumentsHandler`
   - `GET /api/documents/{id}` -> `s.GetDocumentHandler`
   - `DELETE /api/documents/{id}` -> `s.DeleteDocumentHandler`
</action>
<read_first>
- internal/server/server.go
- internal/server/routes.go
</read_first>
<acceptance_criteria>
- Routes are registered under the `/v1` or `/api` prefix (following the project's convention).
- Application compiles with new fields and methods.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- CORS: Ensure the new endpoints are covered by the existing CORS middleware.
- Error reporting: Friendly JSON error messages for common failures (invalid file type, too large, etc.).
- Performance: Uploading should be non-blocking for processing.
</must_haves>

<automated>
- `go test -v ./internal/server -run TestDocumentHandlers` (using `httptest`)
</automated>
