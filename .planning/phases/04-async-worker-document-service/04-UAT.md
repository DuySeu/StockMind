---
status: testing
phase: 04-async-worker-document-service
source: [04-VALIDATION.md]
started: 2026-04-08T03:36:00Z
updated: 2026-04-08T03:36:00Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

number: 1
name: Cold Start Smoke Test
expected: |
  Kill any running server/service. Clear ephemeral state (temp DBs, caches, lock files). Start the application from scratch. Server boots without errors, any DB migration completes, Qdrant connects, and a primary query (health check) returns live data.
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running server/service. Clear ephemeral state (temp DBs, caches, lock files). Start the application from scratch. Server boots without errors, any DB migration completes, Qdrant connects, and a primary query (health check) returns live data.
result: [pending]

### 2. Async processing flow
expected: Upload a standard document (`curl -F file=@sample.pdf ...`). You receive an HTTP 202 Accepted response immediately with a Document ID, while the processing (Parsing, Chunking, Embedding) happens in the background.
result: [pending]

### 3. Status polling
expected: Query the document status via `GET /v1/documents/:id`. The status should successfully transition from `pending` -> `processing` -> `ready`.
result: [pending]

### 4. Document listing
expected: Query `GET /v1/documents`. The list includes the newly processed document with the correct metadata and chunk counts.
result: [pending]

### 5. Document deletion
expected: Call `DELETE /v1/documents/:id`. The document is correctly removed; querying it again returns a 404, and its associated vectors are safely removed from Qdrant.
result: [pending]

### 6. Large file rejection
expected: Attempt to upload an extremely large document (e.g., > 10MB). The server accurately rejects it with an HTTP 413 Request Entity Too Large error without causing memory issues.
result: [pending]

### 7. Error handling
expected: Upload a corrupted or encrypted document. The document status correctly updates to `failed` and exposes a descriptive error message indicating the failure point.
result: [pending]

### 8. Graceful shutdown
expected: Upload a document and immediately send SIGTERM to the server process while processing. The worker intercepts the signal, completes the in-flight job, and only then exits smoothly.
result: [pending]

## Summary

total: 8
passed: 0
issues: 0
pending: 8
skipped: 0

## Gaps

