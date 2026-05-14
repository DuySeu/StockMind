# RAG Ingestion Refactor: MinIO + Elastic Worker + Internal Tool

## Goal

Refactor the RAG ingestion pipeline with three changes:
1. Replace temp file storage with MinIO object storage
2. Replace fixed worker pool with elastic worker pool (0→2, auto-scale, auto-kill on idle)
3. Move `retrieve_knowledge` from MCP to internal tool (direct function call)

## 1. MinIO Storage Layer

### New Package

```
internal/storage/minio.go
```

### Interface

```go
type ObjectStore interface {
    Put(ctx context.Context, key string, reader io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

### Flow

```
POST /v1/documents/
  → Create DB record (status=pending, object_key=documents/{docID}/{filename})
  → Upload file to MinIO
  → Enqueue job (contains objectKey, not temp file path)
  → Worker downloads from MinIO → pipes into IngestPipeline
```

### Key Naming

`documents/{docID}/{original_filename}` — no user-controlled path traversal.

### Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MINIO_ENDPOINT` | `localhost:9000` | MinIO server address |
| `MINIO_ACCESS_KEY` | `minioadmin` | Access key |
| `MINIO_SECRET_KEY` | `minioadmin` | Secret key |
| `MINIO_BUCKET` | `documents` | Bucket name |
| `MINIO_USE_SSL` | `false` | TLS toggle |

### Startup

Bucket auto-created if not exists (idempotent `MakeBucket` with `BucketExists` check).

### Delete Flow

When user deletes document: delete vectors (Qdrant) + delete metadata (PG) + delete object (MinIO).

## 2. Elastic Worker Pool

### Constants

```go
const (
    maxWorkers        = 2
    workerIdleTimeout = 10 * time.Second
)
```

### Behavior

- Queue empty → 0 workers active
- Job enqueued → spawn worker if `activeWorkers < maxWorkers`
- Worker finishes job → check channel → if empty, wait `workerIdleTimeout`
- After timeout with no job → worker exits
- Burst (5 files) → 2 workers process in parallel, 3 queued

### Mechanism

```go
type Worker struct {
    store         ObjectStore
    pipeline      *kb.IngestPipeline
    db            *database.Queries
    jobs          chan *Job
    mu            sync.Mutex
    activeWorkers int
    wg            sync.WaitGroup
}
```

- `Enqueue()`: push to channel + `trySpawn()`
- `trySpawn()`: lock → check < max → increment → unlock → `go w.run()`
- `run()`: loop with `select { case job: process | case <-time.After(idle): exit }`
- Worker exit: lock → decrement → unlock → `wg.Done()`

### Non-blocking Enqueue

```go
func (w *Worker) Enqueue(job *Job) error {
    select {
    case w.jobs <- job:
        w.trySpawn()
        return nil
    default:
        return ErrQueueFull
    }
}
```

Channel buffer: 20. If full → return error to HTTP handler (503).

### Shutdown

1. Close job channel
2. `wg.Wait()` — blocks until in-flight jobs complete
3. Workers detect closed channel → exit after current job

### Job Struct

```go
type Job struct {
    DocID     uuid.UUID
    Name      string
    FileType  string
    Strategy  kb.Strategy
    ObjectKey string
}
```

### Panic Safety

Each worker goroutine: `defer wg.Done()` + `defer recover()` with error logging.

## 3. Internal Tool `retrieve_knowledge`

### Change

Remove from MCP server. Register directly into `ToolManager` inside `NewServer()`.

### Handler

```go
func retrieveKnowledgeHandler(retriever kb.Retriever) core.ToolHandler {
    return func(ctx context.Context, args string) (string, error) {
        var params struct{ Query string `json:"query"` }
        json.Unmarshal([]byte(args), &params)
        results, err := retriever.Search(ctx, params.Query, kb.SearchHybrid, 5)
        // format and return
    }
}
```

### Registration (in NewServer)

```go
toolMgr.Register(mcp.NewTool("retrieve_knowledge", ...), retrieveKnowledgeHandler(retriever))
```

LLM sees the same tool name and schema — no agent flow config changes needed.

### Files Removed

- `internal/mcp/rag_tool.go` — deleted
- `internal/mcp/service.go` — remove `retrieve_knowledge` registration + `retriever` parameter

## Integration Changes

| File | Change |
|------|--------|
| `internal/storage/minio.go` | New: ObjectStore interface + MinIO implementation |
| `internal/common/configs.go` | Add MinIO config struct + env loading |
| `internal/service/worker/worker.go` | Rewrite: elastic pool, download from MinIO |
| `internal/service/service.go` | Inject ObjectStore into Worker |
| `internal/server/document.handler.go` | Upload to MinIO instead of temp file |
| `internal/server/server.go` | Register `retrieve_knowledge` in ToolManager, inject ObjectStore |
| `internal/mcp/service.go` | Remove `retrieve_knowledge` + `retriever` param |
| `internal/mcp/rag_tool.go` | Delete |
| `cmd/main.go` | Init MinIO client, remove retriever from mcp.Start() |
| `.env.example` | Add MINIO_* vars |
| `docker-compose.yml` | Already has MinIO (no change) |

## Error Handling

- MinIO upload fail → return 500 to user, no DB record orphan (DB first, upload second pattern: if upload fails, mark document as `failed`)
- Worker download fail (object not found) → mark job `failed` in DB
- Queue full → return 503 to HTTP handler
- Worker panic → recover, log, mark job `failed`, worker exits cleanly

## Retry Strategy

Files persist on MinIO. On startup, scan DB for `status=processing` documents (crashed mid-process) → re-enqueue. Future enhancement, not in initial implementation.

## Testing

- `internal/storage/` — unit test with mock or testcontainers MinIO
- `internal/service/worker/` — test elastic spawn/kill behavior with fake jobs
- Internal tool handler — mock `kb.Retriever`, verify JSON parsing + output format
