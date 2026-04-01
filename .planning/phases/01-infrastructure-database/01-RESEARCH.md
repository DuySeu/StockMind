# Phase 1 Research: Infrastructure & Database

**Phase:** 1 — Infrastructure & Database
**Researched:** 2026-04-01
**Goal:** Qdrant chạy trong Docker, bảng `documents` tồn tại trong PostgreSQL, backend kết nối được với Qdrant khi startup.

---

## Codebase Findings

### docker-compose.yml Pattern
- **Cả 2 files** (`docker-compose.yml` và `docker-compose.dev.yml`) cùng pattern: service với healthcheck, volume, network `stockmind`
- Healthcheck pattern cho postgres: `CMD-SHELL + pg_isready`, interval=5s, timeout=5s, retries=3, start_period=15s
- Qdrant healthcheck: dùng `curl -f http://localhost:6333/readyz || exit 1` (Qdrant có built-in health endpoint)
- **Cả 2 files đều cần add Qdrant** — dev và prod phải nhất quán
- `app` service depends_on db với `condition: service_healthy` — cần làm tương tự cho Qdrant

### Goose Migration Pattern
- File: `schema/migrations/0001_init.sql`
- Format: `-- +goose Up` header bắt buộc
- Dùng `-- +goose statementbegin / statementend` cho stored procedures
- Không có `-- +goose Down` trong file hiện tại (thêm vào cho completeness)
- **Next migration number:** `0002_add_documents.sql`
- Trigger function `trigger_set_timestamp()` đã tồn tại → dùng lại cho documents table

### sqlc Pattern
- **Config:** `sqlc.yaml` → schema: `schema/migrations`, queries: `schema/queries`, output: `internal/database`
- **Query format:** `-- name: FunctionName :one/:many/:exec` annotation
- **UUID type:** dùng `github.com/google/uuid.UUID` (đã configured trong sqlc.yaml overrides)
- **pgx/v5 driver** — tất cả queries dùng pgx interface
- Pattern: `RETURNING *` cho INSERT queries (`:one`)
- Running sqlc: `sqlc generate` (no Makefile target hiện tại — cần add hoặc run trực tiếp)

### Existing Tables (từ 0001_init.sql)
```sql
users, agent_flows, sessions, session_history, research, watchlist, news
```
Tất cả đều có: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `created_at`, `updated_at` với trigger

### Go Module & Dependencies
- Module: `stockmind`
- **Qdrant client chưa có** trong go.mod → cần `go get github.com/qdrant/go-client`
- `golang.org/x/sync` đã có (dùng cho errgroup) ✓
- `github.com/sethvargo/go-retry` đã có (retry logic) ✓

### Internal Package Structure
```
internal/
  agent/      — AI agent lifecycle
  common/     — shared utilities
  database/   — sqlc generated code
  mcp/        — MCP tools/server
  server/     — HTTP handlers + routing
  service/    — business logic (currently only tavily/)
```
→ Sẽ add `internal/rag/` package ở Phase 2-5, Phase 1 chỉ cần `internal/database/` (sqlc generated)

### .env Configuration
- Current: PORT, APP_ENV, DB_*, OPENROUTER_API_KEY, TAVILY_API_KEY
- **Cần add:** `QDRANT_HOST`, `QDRANT_PORT` (default: localhost, 6334)
- `QDRANT_HOST` default `localhost` cho dev, `stockmind_qdrant` cho Docker networking

---

## Qdrant Go Client API

### Package
```go
import (
    "github.com/qdrant/go-client/qdrant"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)
```

### Connection
```go
conn, err := grpc.NewClient(
    "localhost:6334",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
client := qdrant.NewCollectionsClient(conn)
```

### Collection Creation (idempotent)
```go
_, err = client.Create(ctx, &qdrant.CreateCollection{
    CollectionName: "stockmind_knowledge",
    VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
        Size:     2048, // nvidia/llama-nemotron-embed-vl-1b-v2 output dim
        Distance: qdrant.Distance_Cosine,
    }),
})
// If already exists → gRPC status code AlreadyExists → ignore
```

### Check Collection Exists
```go
pointsClient := qdrant.NewPointsClient(conn)
collectionsClient := qdrant.NewCollectionsClient(conn)
resp, err := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
    CollectionName: "stockmind_knowledge",
})
```

### Retry Pattern (using sethvargo/go-retry — already in go.mod)
```go
import "github.com/sethvargo/go-retry"

err = retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
    if err := ping(ctx); err != nil {
        return retry.RetryableError(err)
    }
    return nil
})
```
Max 5 attempts → backoff capped at 16s (fibonacci sequence: 1,1,2,3,5,8,13... cap at 16)

---

## documents Table Schema Design

```sql
CREATE TABLE IF NOT EXISTS documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(500) NOT NULL,
    file_type   VARCHAR(10) NOT NULL CHECK (file_type IN ('pdf','docx','md','txt')),
    size_bytes  BIGINT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','processing','ready','failed')),
    chunk_count INT4 NOT NULL DEFAULT 0,
    strategy    VARCHAR(20) NOT NULL DEFAULT 'recursive'
                  CHECK (strategy IN ('recursive','fixed','paragraph','semantic')),
    error_msg   TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TRIGGER set_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();
```

No `-- +goose Down` needed but good to add:
```sql
-- +goose Down
DROP TABLE IF EXISTS documents;
```

---

## sqlc Queries for documents

```sql
-- name: CreateDocument :one
INSERT INTO documents (id, name, file_type, size_bytes, strategy)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = $1;

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at DESC;

-- name: UpdateDocumentStatus :exec
UPDATE documents
SET status = $2, chunk_count = $3, error_msg = $4
WHERE id = $1;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE id = $1;
```

---

## Validation Architecture

### How to verify correctness for each deliverable:

1. **Qdrant Docker:**
   - `curl http://localhost:6333/readyz` → `{"result":true}`
   - `curl http://localhost:6333/dashboard` → HTTP 200

2. **Migration:**
   - `psql -c "\d documents"` → shows columns with correct types/constraints
   - `psql -c "INSERT INTO documents(name,file_type,size_bytes,strategy) VALUES('test','pdf',1000,'recursive') RETURNING id"` → returns UUID

3. **sqlc generate:**
   - `internal/database/documents.sql.go` exists and compiles
   - Functions: `CreateDocument`, `GetDocumentByID`, `ListDocuments`, `UpdateDocumentStatus`, `DeleteDocument`

4. **Qdrant client init:**
   - Backend log contains: `"Qdrant collection ready: stockmind_knowledge"`
   - `curl http://localhost:6333/collections/stockmind_knowledge` → `"status":"green"`, `"vectors_config":{"params":{"size":2048}}`

5. **Retry behavior:**
   - Start backend before Qdrant → log shows retry attempts → eventually connects when Qdrant starts

---

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Add Qdrant to BOTH docker-compose.yml and docker-compose.dev.yml | Dev/prod parity; dev compose is the primary workflow |
| `QDRANT_HOST` env var with default `localhost` | Allows override for Docker networking (use service name) |
| Qdrant gRPC port 6334 (not REST 6333) | Go client uses gRPC; REST port exposed for dashboard/debugging only |
| `sethvargo/go-retry` for Qdrant retry | Already in go.mod (transitive dep), consistent with codebase |
| Qdrant client init in new `internal/rag/client.go` | Separation: `internal/rag/` package for all RAG concerns |
| `documents` table no foreign key to `users` | Phase 1 spec: shared global knowledge base, no per-user isolation |
