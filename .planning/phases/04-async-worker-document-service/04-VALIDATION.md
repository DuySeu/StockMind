# Phase 4 Verification: Async Worker & Document Service + REST API

## UAT Criteria

| ID | Goal | Test Method | Expected Result |
|----|------|-------------|-----------------|
| UAT-4-01 | Large file rejection | `curl -F file=@11MB.pdf ...` | HTTP 413 Request Entity Too Large |
| UAT-4-02 | Async processing flow | `curl -F file=@sample.pdf ...` | HTTP 202 Accepted, ID returned |
| UAT-4-03 | Status polling | `GET /v1/documents/:id` | Status transitions: `pending` -> `processing` -> `ready` |
| UAT-4-04 | Document listing | `GET /v1/documents` | List includes the new document with correct metadata |
| UAT-4-05 | Document deletion | `DELETE /v1/documents/:id` | HTTP 200/204; record is GONE from both DB and search results |
| UAT-4-06 | Error handling | Upload corrupt/encrypted PDF | Status becomes `failed` with descriptive error message |
| UAT-4-07 | Graceful shutdown | Upload large PDF then SIGTERM | Worker completes the current job before the process exits |

## Technical Checklist

- [ ] Worker pool initialized with size 2.
- [ ] Buffered channel capacity 10.
- [ ] Temporary files cleaned up in all code paths (success/fail).
- [ ] Document status updated to `failed` on worker errors.
- [ ] Chi routes correctly configured under `/v1/documents`.
- [ ] Embeddings batch size = 20.
- [ ] UUIDs used for all document IDs.

## Regression Testing

- [ ] Chat tool `retrieve_knowledge` still works (if Phase 5 already implemented, otherwise skip).
- [ ] Database migration `documents` table existence (should be already present from Ph 1).
- [ ] Server startup time with Qdrant connectivity checks.
