# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

StockMind is an AI financial assistant for the Vietnamese stock market: a Go monolith (chi HTTP server + agentic LLM loop + in-process tools + hybrid RAG) serving a React 19 SPA. Backends: PostgreSQL, Qdrant (vectors), MinIO (document files).

> **Docs drift warning:** `AGENTS.md`, `README.md`, and `GEMINI.md` are partly stale. They describe a standalone `mcp` subcommand and `internal/agent/`, `internal/rag/`, `internal/knowledge_base/` packages that no longer exist. Trust the code and this file. See "Recent refactor" below.

## Commands

```bash
make run            # go run cmd/main.go server (background) + Vite dev server (frontend)
make build          # go build -o main cmd/main.go
make test           # go test ./... -v
make itest          # DB integration tests (go test ./internal/database -v)
make watch          # live reload via air (installs air if missing)
make docker-build   # build duy0207/stockmind:latest image

go test ./internal/knowledge -run TestName   # run a single test (knowledge is the only package with tests)
sqlc generate       # regenerate internal/database/*.sql.go from schema/queries/*.sql
cd frontend && npm run dev | npm run build | npm run lint
```

The only CLI subcommand is `server` (flag: `--port`, default 8080). There is **no** `mcp` subcommand — the `--mcp-protocol` flag in `.air.toml` is stale and ignored.

Infra (`docker-compose up -d`) starts PostgreSQL 17, Qdrant, MinIO. DB migrations (goose) run automatically on startup via `database.MigrateDB`.

## Architecture

Entry point `cmd/main.go` → `runServer()` wires everything in order: DB pool → migrate → MinIO → knowledge base (Qdrant + embedder + BM25) → external MCP client manager → services (worker pool) → tools → LLM service → HTTP server. `runServer` returns a `shutdown()` closure that drains HTTP, then worker pool, then MCP sessions.

Request path: **React SPA → chi HTTP (`internal/server`) → `LLMService` agentic loop (`internal/llm`) → tools (`internal/tools`) + provider (`internal/llm/providers`) → Postgres/Qdrant/MinIO.**

### Tools — the central concept

There is **no standalone MCP server anymore.** Tools come from two sources and are merged into one `tools.Manager`:

1. **In-process Go tools** (`internal/tools/implementations/`): `retrieve_knowledge`, `get_stock_price`, `get_report`, `piotroski_evaluation`, `get_news`, `fundamental_analysis`. Registered in `tools.RegisterTools()` (`tool_registry.go`), which injects deps (KB retriever, services) via closures. Each uses the generic `NewTool[In](...)` — JSON Schema is inferred from struct tags (`json` + `jsonschema`), or from a custom `Schema()` method if the input implements `SchemaProvider`.
2. **Bridged external MCP tools** (`internal/mcp` + `tools.BridgeMCPTools`): `internal/mcp` is now an MCP **client** manager, not a server. At startup it lazily spawns configured external MCP servers (currently AWS docs via `uvx awslabs.aws-documentation-mcp-server`, skipped if `uvx` is absent), lists their tools, and wraps each as a local `*Tool` named `<server>_<tool>`. Calls route through `Manager.CallTool`, which evicts and reconnects once on failure.

To add a native tool: write a handler in `implementations/`, add its input struct + `Handle*` func, register it in `RegisterTools`.

### VietCap data sources

VietCap's GraphQL endpoint (`trading.vietcap.com.vn/data-mt/graphql`) is **decommissioned** — it answers every query with `{}`, including invalid ones. `get_report` and `piotroski_evaluation` now read the IQ REST service (`common.IQ_INSIGHT_URL`) via `common.FetchIQInsight`, which is the single choke point for that service and rejects an empty payload rather than returning a zero value. Price board and OHLC chart still come from `trading.vietcap.com.vn/api`.

Live canaries for these endpoints live behind a build tag: `go test -tags integration ./internal/...`. They fail by name if VietCap changes shape again.

### LLM providers

`LLMService` (`internal/llm/service.go`) is DB-agnostic (caller persists). Provider chosen at startup from `LLM_PROVIDER` (`openai` | `anthropic` | `openrouter`); each is wrapped into a `completionFunc` (streaming, returns `<-chan database.StreamEvent`) and a `structuredCompletionFunc` (non-streaming JSON → struct). Model from `LLM_MODEL`. **By default all three providers point at OpenRouter's base URL** (see `LoadConfig`), so the "provider" mostly selects which SDK/streaming shape is used. Anthropic additionally supports AWS Bedrock via assume-role (commented out in config). All non-Bedrock SDKs share `common.SharedHTTPClient` for connection reuse.

The agentic loop: call LLM → stream deltas → if tool calls, execute via `Manager.Execute` → append results → repeat until done. `internal/llm/summarizer.go` compresses long histories to save tokens.

### Hybrid RAG (`internal/knowledge`)

`kb.New()` builds Qdrant store + OpenRouter embedder + BM25. Documents: dense vectors (via `EMBED_MODEL`) **and** sparse BM25 vectors; retrieval fuses both with Reciprocal Rank Fusion (RRF). Ingestion is async: upload → MinIO → worker pool job (`internal/service/worker`, elastic, max ~2 workers) → parse (PDF/DOCX/MD/TXT) → chunk → embed → BM25 → Qdrant upsert → update DB status.

### Data layer

sqlc, SQL-first. Edit `schema/queries/*.sql` (and `schema/migrations/` for DDL), then `sqlc generate` → `internal/database/*.sql.go`. Never hand-edit generated `*.sql.go`. JSONB columns map to custom Go types via `overrides` in `sqlc.yaml` (`internal/database/types.go`: `StreamEvent`, `AgentFlowConfig`, `StockReport`, `[]Metadata`).

### Prompts

System/task prompts are text templates in `internal/llm/prompts/templates/*.txt`, loaded via `internal/llm/prompts/loader.go`. Edit the `.txt` files to change LLM behavior.

## Conventions

- Config is env-only (`common.LoadConfig` reads `os.Getenv`); `.env` loaded via godotenv. Required: DB_*, `OPENROUTER_API_KEY`, `TAVILY_API_KEY`, `LLM_PROVIDER`, `LLM_MODEL`, `EMBED_MODEL`. Optional: `QDRANT_*`, `MINIO_*`, `PORT`.
- HTTP handlers are methods on `*Server` in `internal/server/*.handler.go`, wired in `routes.go`. Chat streams SSE directly (`start`/`thinking`/`text`/`tool_call`/`tool_result`/`error`/`done` events) — no pub/sub layer.
- `.cursor/rules/` documents team practices: incremental (thin vertical slices, test each), TDD, and code-review discipline.

## Known gaps

No auth (hardcoded default user UUID). WebSocket handler exists but route is commented out. Frontend login is non-functional. Test coverage is limited to `internal/knowledge`. `agent_flows` configs are seeded/stored but the chat path uses the single env-configured provider/model.

## Recent refactor (uncommitted at time of writing)

`internal/agent/` → `internal/llm/`; `internal/knowledge_base/` → `internal/knowledge/`; the 5 MCP financial tools became in-process handlers under `internal/tools/implementations/`; `internal/mcp` flipped from server to client/bridge; `fundamental_analysis` tool + endpoint added.
