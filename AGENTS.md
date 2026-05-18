# AGENTS.md — StockMind

> AI-powered financial assistant for the Vietnamese stock market. Go backend + React frontend + MCP tool server + hybrid RAG pipeline.

## Directory Map

```
cmd/main.go                     → CLI entry (urfave/cli/v3): `server` or `mcp` subcommands
internal/
  llm/                          → LLM orchestration: multi-provider (OpenRouter/OpenAI/Anthropic),
                                  session-aware chat loop, tool execution via ToolManager
    service.go                  → LLMService: Chat() (session-aware) + CallLLM() (stateless agentic loop)
    tool_manager.go             → Thread-safe tool registry + dispatch
    openrouter.go               → OpenRouter SDK streaming + embeddings
    openai.go                   → OpenAI provider
    anthropic.go                → Anthropic provider (supports AWS Bedrock)
  server/                       → HTTP layer: chi router, REST handlers, direct SSE streaming
    server.go                   → Server struct, NewServer(), registers retrieve_knowledge tool
    routes.go                   → All route definitions + chatHandler (SSE) + SPA serving
    researcher.handler.go       → Market research: Tavily + LLM digest → persisted reports
    document.handler.go         → Document upload/list/get/delete
    stock.handler.go            → Price board, watchlist, research reports
    session.handler.go          → Session list/get/delete
    news.handler.go             → News endpoint
    user.handler.go             → User CRUD
    agent_flow.handler.go       → Agent flow list/create
  mcp/                          → MCP tool server (standalone): 5 financial analysis tools
  knowledge_base/               → Hybrid RAG: parse → chunk → embed → Qdrant (dense + sparse + RRF)
    ingest.go                   → IngestPipeline: parse → chunk → embed → BM25 → upsert
    store.go                    → QdrantStore: dense (2048-dim cosine) + sparse (BM25) vectors
    retriever.go                → Hybrid retriever: dense + sparse + RRF fusion
    bm25.go                     → BM25 tokenizer (30K vocab)
    chunker.go                  → 4 strategies: semantic, fixed, sentence, paragraph
    parser.go                   → PDF, DOCX, MD, TXT parsers
    embedder.go                 → OpenRouter embeddings
  database/                     → sqlc-generated queries + custom types (StreamEvent, AgentConfig, etc.)
  service/                      → Worker pool (elastic, max 2, queue 20) + Tavily client
  storage/                      → MinIO object store (document files before processing)
  common/                       → Config loading, VietCap API constants, JSON/SSE response helpers
frontend/src/
  pages/                        → Chatbot, HomePage, WatchList, MarketResearcherPage, DocumentPage,
                                  ResearchResultPage, SettingPage, LoginPage
  api/                          → Axios + fetch clients: chat (SSE), stock, documents, sessions, news
  components/                   → MessageList, SideBar, ResearchReport + shadcn/ui primitives
schema/
  migrations/                   → Goose SQL migrations (single 0001_init.sql)
  queries/                      → sqlc SQL definitions
  knowledge_base/               → Static financial data (CSV, PDF, XLSX)
```

## Architecture Overview

Layered monolith: React SPA → chi HTTP server → LLM service → providers (OpenRouter/OpenAI/Anthropic) + MCP tools → PostgreSQL + Qdrant + MinIO.

```
Browser (React SPA)
    ↓ REST + SSE (direct streaming)
Chi HTTP Server (Go)
    ↓
LLM Service (agentic tool loop)
    ├── Internal tools (retrieve_knowledge)
    ├── MCP Client → MCP Tool Server (financial tools)
    └── LLM Providers (via OpenRouter)
    ↓
PostgreSQL (sessions, reports, documents, users, watchlist)
Qdrant (dense + sparse vectors for RAG)
MinIO (document file storage)
```

Docker Compose runs PostgreSQL 17, Qdrant, and MinIO. The backend serves `frontend/dist/` in production.

## Key Patterns

**LLM Provider Selection**: Provider is configured via `LLM_PROVIDER` env var (openrouter/openai/anthropic). All providers can use OpenRouter as base URL. Model set via `LLM_MODEL` env var.

**Direct SSE Streaming**: `chatHandler` (POST `/v1/chat`) sets SSE headers, sends a "start" event with session_id, then streams events directly from the LLM channel to the HTTP response. No intermediate pub/sub or StreamManager.

**Agentic Tool Loop**: `LLMService.CallLLM()` runs a stateless loop — call LLM → if tool calls → execute via ToolManager → append results → call LLM again. Continues until done or error.

**Internal vs MCP Tools**: `retrieve_knowledge` is registered directly in `server.go` as a ToolManager handler (not in the MCP server). The 5 financial tools live in the MCP server.

**Hybrid RAG**: Documents are embedded with both dense vectors (2048-dim via OpenRouter) and sparse vectors (BM25). Retrieval uses Reciprocal Rank Fusion (RRF) to combine results from both.

**Document Processing Pipeline**: Upload → MinIO storage → worker pool job → parse → chunk → embed → Qdrant upsert. Worker pool is elastic (max 2 workers, queue 20, 10s idle timeout).

**sqlc Code Generation**: All database queries are SQL-first in `schema/queries/`, generated to Go with `sqlc generate`. Custom types in `internal/database/types.go` handle JSONB columns.

**Agent Flow Configs**: Multi-agent pipeline configs stored as JSONB in `agent_flows` table. Currently seeded in migration but not actively used in the chat path — the chat uses the single configured provider/model.

## MCP Tools

5 tools registered in the standalone MCP server (`internal/mcp/service.go`):

| Tool | What it does |
|------|-------------|
| `get_stock_price` | OHLC data from VietCap (symbol, time_frame, count_back) |
| `piotroski_evaluation` | 9-point Piotroski F-Score from financial statements |
| `altman_z_score` | Altman Z-Score bankruptcy predictor |
| `get_report` | Quarterly/yearly financial reports (symbol, period) |
| `get_news` | Stock news via Tavily search (query) |

Plus 1 internal tool registered in `server.go`:

| Tool | What it does |
|------|-------------|
| `retrieve_knowledge` | Hybrid RAG search (dense + sparse + RRF) on Qdrant |

MCP server supports stdio and HTTP (streamable HTTP on :8081) protocols.

## Data Flow

**Chat**: POST `/v1/chat` → create/reuse session → load history → LLM agentic loop (completion → tool calls → tool results, repeats until done) → stream SSE events directly to client.

**SSE Events**: `start` (session_id), `thinking` (delta), `text` (delta), `tool_call` (name + args), `tool_result` (result), `error`, `done`.

**Research**: POST `/v1/stock/research` → for each ticker: Tavily web research → LLM digest → StockReport JSON → persist to DB. Also available as SSE stream via POST `/v1/stock/research/stream`.

**RAG Ingest**: POST `/v1/documents/` → save metadata to DB → upload file to MinIO → enqueue worker job → worker: download from MinIO → parse → chunk → embed (dense) → BM25 vectorize (sparse) → Qdrant upsert → update DB status.

**RAG Retrieval**: During chat, `retrieve_knowledge` tool → hybrid search (dense + sparse prefetch → RRF fusion) → return formatted chunks to LLM.

## Configuration

**Required env vars**: `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `OPENROUTER_API_KEY`, `TAVILY_API_KEY`, `LLM_PROVIDER`, `LLM_MODEL`, `EMBED_MODEL`.

**Optional**: `PORT` (default 8080), `QDRANT_HOST`, `QDRANT_PORT`, `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`.

**CLI**: `go run cmd/main.go server [--port 8080] [--mcp-protocol http|stdio]` or `go run cmd/main.go mcp`.

## Known Gaps

- **No authentication** — hardcoded default user UUID, all endpoints unauthenticated
- **Agent flows unused** — configs exist in DB but chat uses single provider/model from env vars
- **No structured error handling** — mixed `WriteJSONError` and `http.Error` usage
- **Limited tests** — only knowledge_base package has tests; no handler, LLM, or frontend tests
- **WebSocket disabled** — handler exists but route is commented out
- **Login non-functional** — LoginPage UI exists but form does nothing
- **Migration trigger bug** — `set_documents_updated_at` trigger is created then immediately dropped
- **Mixed i18n** — Vietnamese and English text mixed throughout frontend

## Custom Instructions
<!-- This section is for human and agent-maintained operational knowledge.
     Add repo-specific conventions, gotchas, and workflow rules here.
     This section is preserved exactly as-is when re-running codebase-summary. -->
