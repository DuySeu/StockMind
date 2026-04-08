---
status: human_needed
phase: 4-async-worker-document-service
started: 2026-04-08T04:00:00Z
updated: 2026-04-08T04:00:00Z
---

# Phase 04 Verification

## Goal Achievement
**Target:** Upload document qua API → xử lý async → status tracking hoạt động đầy đủ.
**Status:** ✅ Implemented. The endpoints `POST /v1/documents` handle uploads with a 10MB limit and interact with `DocumentService`. `DocumentService.Upload` writes the metadata properly mapped with UUIDs to Postgres and enqueues the `*rag.Job` on a bounded background queue. Goroutines from `Worker.Start()` asynchronously parse, chunk, embed, and store into Qdrant, keeping the SQL tracking up-to-date locally.

## Must-Haves Check
- **Must-Have 1:** Graceful shutdown of active workers. (✅ `worker.wg.Wait()` covers this during graceful shutdown of main app context)
- **Must-Have 2:** Fail-safe file handles and status consistency. (✅ `defer os.Remove(job.TempFile)` is present; statuses mapped across `pending` → `processing` → `ready`/`failed` in `worker.process`)
- **Must-Have 3:** Document records and orphaned vectors properly deleted on document deletion. (✅ `DocumentService.Delete` performs store and DB unlinks)

## Automated Checks
- Automated validations depend on integration pipelines missing from the direct code coverage (Wait to be verified by human E2E check).

## Human Verification Required

1. **Test Async Document Upload**
   - **Action:** Using curl or Postman, upload a test PDF to `POST /v1/documents`.
   - **Expected:** Immediately receive a `202 Accepted` response with the status `pending`.

2. **Test Processing Status Change**
   - **Action:** Wait a few seconds, then query `GET /v1/documents/{id}` using the UUID from the response above.
   - **Expected:** The status transitions efficiently from `processing` to `ready`, with `chunk_count` matching the actual chunk payload.

3. **Test Graceful File Restrictions**
   - **Action:** Try to upload a single 15MB file.
   - **Expected:** Receive a `413 Request Entity Too Large` failure directly from `http.MaxBytesReader`.
