---
plan: 03/03-STORAGE
status: complete
phase: 03
wave: 2
---

# Plan 03 Summary: Qdrant Storage Layer

## What was built

- **`store.go`** — `Store` interface (`Upsert`, `Delete`) and `QdrantStore` implementation.

## Key implementation details

### Payload schema (D-09 from CONTEXT.md)
Each chunk point contains:
```json
{
  "doc_id": "<uuid string>",
  "chunk_index": 0,
  "text": "<chunk text>",
  "strategy": "recursive"
}
```

### Idempotent re-indexing
Chunk point IDs are deterministic: `uuid.NewSHA1(namespace=docID, data=chunkIndex)`. Re-uploading the same document will overwrite existing points (Qdrant upsert semantics).

### Model consistency check (D-10 from CONTEXT.md)
A sentinel config point (id=0) stores `{embedding_model: "...", _type: "config"}` in payload. On startup:
- First run → write sentinel.
- Subsequent runs → compare stored model with `embeddingModel` constant; return error if mismatch.
- Real chunk searches must filter `chunk_index >= 0` (Phase 5 will do this) to exclude the sentinel.

### Delete
Uses Qdrant `Filter.Must[Match("doc_id", docID)]` to bulk-delete all chunks for a document without knowing individual IDs.

## self-check: PASSED
- `go build ./...` — clean (full codebase)
- `TestStore_UpsertLengthMismatch, TestStore_UpsertEmptyDocID, TestStore_UpsertEmptyChunks, TestStore_DeleteEmptyDocID, TestStoreConsistency_ConfiguredModel` — 5/5 PASS
- Full suite: 27/27 tests PASS

## key-files.created
- `internal/rag/store.go`
- `internal/rag/store_test.go`
