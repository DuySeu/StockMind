# AGENTS.md — StockMind

> AI-powered financial assistant for the Vietnamese stock market. Go backend + React frontend + MCP tool server + RAG pipeline.

## Table of Contents
<!-- tags: navigation -->

- [Directory Map](#directory-map) — where to find things
- [Architecture Overview](#architecture-overview) — how components connect
- [Key Patterns](#key-patterns) — non-obvious design decisions
- [MCP Tools](#mcp-tools) — financial analysis tools available to agents
- [Data Flow](#data-flow) — how chat, research, and RAG work
- [Configuration](#configuration) — env vars and agent flow configs
- [Known Gaps](#known-gaps) — what's missing or incomplete
- [Detailed Documentation](#detailed-documentation) — deep-dive reference files
- [Custom Instructions](#custom-instructions) — human-maintained conventions

## Directory Map
<!-- tags: navigation, structure -->

```
cmd/main.go                     → CLI entry: `server` (HTTP + MCP) or `mcp` (standalone)
internal/
  agent/                        → LLM orchestration: dual-provider (OpenAI/Anthropic via OpenRouter),
                                  session turn loop, MCP client integration, config-driven agent flows
  server/                       → HTTP layer: chi router, REST handlers, SSE streaming (pub/sub)
    routes.go                   → All route definitions + chat handler + SSE handler
    researcher.handler.go       → Market research: Tavily + LLM digest → persisted reports
    stream.go                   → StreamManager for SSE event buffering
  mcp/                          → MCP tool server: stock prices, Piotroski, Altman Z, reports, news, RAG
  rag/                          → RAG pipeline: parse (PDF/DOCX/MD/TXT) → chunk → embed → Qdrant
  database/                     → sqlc-generated queries + custom types (AgentFlowConfig, MessageUnion)
  service/                      → DocumentService, Tavily client (web search + async research)
  common/                       → VietCap API constants, response helpers
frontend/src/
  pages/                        → Chatbot, WatchList, MarketResearcher, Documents, Settings, Home, Login
  api/                          → Axios clients: chat, stock, documents, sessions, news, agent_flows
  components/ui/                → shadcn/ui (Radix primitives)
schema/
  migrations/                   → Goose SQL migrations (single 0001_init.sql)
  queries/                      → sqlc SQL definitions
  knowledge_base/               → Static financial data (CSV, PDF, XLSX)
```

## Architecture Overview
<!-- tags: architecture -->

Layered monolith: React SPA → chi HTTP server → agent orchestration → LLM providers (via OpenRouter) + MCP tools → PostgreSQL + Qdrant.

The backend serves the frontend from `frontend/dist/` in production. Docker Compose runs PostgreSQL 17 + Qdrant alongside the app container.

See [.agents/summary/architecture.md](.agents/summary/architecture.md) for full diagrams.

## Key Patterns
<!-- tags: patterns, conventions -->

**Dual LLM via OpenRouter**: Both `AnthropicProvider` and `OpenAIProvider` use OpenRouter as base URL. Model switching happens in agent flow JSON config, not code. Free-tier models defined in `internal/agent/core.go`.

**Config-driven agent flows**: Multi-agent pipelines stored as JSONB in `agent_flows` table. Each flow defines named agents (with provider, model, system prompt, MCP servers) and a node graph (start → agent → agent → end). No code changes needed to add new flows.

**SSE streaming with history replay**: `chatHandler` (POST) returns `session_id` immediately, processes in a goroutine. `chatStreamHandler` (GET) subscribes to a `StreamManager` that buffers events so late-connecting clients get full history.

**MCP tool namespacing**: Tool names are prefixed with MCP server name (e.g., `stocks-mcp--get_stock_price`) to avoid collisions across multiple MCP servers.

**sqlc code generation**: All database queries are SQL-first in `schema/queries/`, generated to Go with `sqlc generate`. Custom types in `internal/database/types.go` handle JSONB columns.

**RAG async worker**: Document uploads are enqueued to a goroutine pool. Processing: parse → chunk (4 strategies) → embed (OpenRouter) → upsert to Qdrant. Status tracked in `documents` table.

## MCP Tools
<!-- tags: mcp, tools -->

| Tool | What it does |
|------|-------------|
| `get_stock_price` | OHLC data from VietCap (configurable time frame + lookback) |
| `piotroski_evaluation` | 9-point Piotroski F-Score from financial statements |
| `altman_z_score` | Altman Z-Score bankruptcy predictor |
| `get_report` | Quarterly/yearly financial reports |
| `get_news` | Stock news via Tavily search |
| `retrieve_knowledge` | RAG retrieval from Qdrant vector store |

MCP server runs on port 8081 (HTTP) or stdio. See [.agents/summary/interfaces.md](.agents/summary/interfaces.md) for parameters.

## Data Flow
<!-- tags: workflows -->

**Chat**: POST `/v1/chat` → create session → agent turn loop (LLM completion → tool calls → tool results, max 10 loops) → SSE stream events to client.

**Research**: POST `/v1/stock/research` → for each ticker: Tavily async research → LLM digest → StockReport JSON → persist to DB.

**RAG**: POST `/v1/documents/` → enqueue → worker: parse → chunk → embed → Qdrant upsert. Retrieved during chat via `retrieve_knowledge` MCP tool.

See [.agents/summary/workflows.md](.agents/summary/workflows.md) for sequence diagrams.

## Configuration
<!-- tags: config, env -->

Required env vars: `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `OPENROUTER_API_KEY`, `TAVILY_API_KEY`. Optional: `PORT` (default 8080), `QDRANT_HOST`, `QDRANT_PORT`.

Agent flow configs are seeded in the migration SQL (`schema/migrations/0001_init.sql`). Three default flows: OpenAI Flow, Anthropic Flow, Multiple Agents Flow.

## Known Gaps
<!-- tags: gaps, todo -->

- **No authentication** — hardcoded default user UUID, all endpoints unauthenticated
- **No structured error handling** — mixed `WriteJSONError` and `http.Error` usage
- **Limited tests** — only RAG package has tests; no handler, agent, or frontend tests
- **WebSocket disabled** — handler exists but route is commented out
- **Agent flow management** — only List/Create; no Update/Delete/UI
- **README outdated** — says Go 1.21+ (actual: 1.25.1), mentions Python/vnstock3 (removed)

## Detailed Documentation
<!-- tags: reference -->

Full documentation in `.agents/summary/`:

| File | Content |
|------|---------|
| [index.md](.agents/summary/index.md) | Documentation index with AI assistant usage guide |
| [codebase_info.md](.agents/summary/codebase_info.md) | Tech stack, directory structure, env vars |
| [architecture.md](.agents/summary/architecture.md) | System diagrams, layer architecture, design patterns |
| [components.md](.agents/summary/components.md) | Per-package component breakdown |
| [interfaces.md](.agents/summary/interfaces.md) | REST API, Go interfaces, MCP tools, SSE events |
| [data_models.md](.agents/summary/data_models.md) | DB schema, Go types, TypeScript types |
| [workflows.md](.agents/summary/workflows.md) | Chat, research, RAG, agent orchestration flows |
| [dependencies.md](.agents/summary/dependencies.md) | All dependencies with versions and purposes |
| [review_notes.md](.agents/summary/review_notes.md) | Documentation gaps and recommendations |

## Custom Instructions
<!-- This section is for human and agent-maintained operational knowledge.
     Add repo-specific conventions, gotchas, and workflow rules here.
     This section is preserved exactly as-is when re-running codebase-summary. -->
