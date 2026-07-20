# Project Structure & Conventions

## Directory Layout
- `cmd/main.go` — CLI entry point (urfave/cli/v3: server, mcp subcommands)
- `internal/llm/` — LLM orchestration, providers, session management, tool dispatch
- `internal/server/` — HTTP handlers, chi routing, direct SSE streaming
- `internal/mcp/` — Standalone MCP tool server (5 financial tools)
- `internal/knowledge_base/` — Hybrid RAG pipeline (parse, chunk, embed, BM25, Qdrant)
- `internal/database/` — sqlc-generated code + custom types (StreamEvent, AgentConfig, etc.)
- `internal/service/` — Worker pool (elastic, max 2) + Tavily client
- `internal/storage/` — MinIO object store for document files
- `internal/common/` — Config loading, VietCap constants, JSON/SSE response helpers
- `schema/queries/` — SQL-first query definitions (run `sqlc generate` after changes)
- `schema/migrations/` — Goose migrations (single 0001_init.sql)
- `frontend/src/` — React 19 SPA (Vite, TypeScript, shadcn/ui)

## Patterns
- Single LLM provider per instance: configured via `LLM_PROVIDER` + `LLM_MODEL` env vars
- Agentic tool loop: LLM → tool calls → execute → append results → LLM again (until done)
- `retrieve_knowledge` is an internal tool registered in server.go (not in MCP server)
- MCP tools use plain names (no namespacing)
- Direct SSE streaming from LLM channel to HTTP response (no StreamManager)
- Document pipeline: upload → MinIO → worker pool → parse → chunk → embed → Qdrant
- Hybrid RAG: dense (2048-dim) + sparse (BM25) + RRF fusion

## Database
- All queries defined in `schema/queries/*.sql`, generated with `sqlc generate`
- Custom types for JSONB columns in `internal/database/types.go`
- Migrations via goose in `schema/migrations/`
- Tables: users, conversations, messages, research, watchlist, news, documents, agent_flows
