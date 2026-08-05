# StockMind Project Documentation

## 1. Overview
StockMind is an AI-powered financial assistant tailored for the Vietnamese stock market. It aims to streamline the investment research process by providing intuitive access to financial data, intelligent analysis, and real-time market insights. The project uses a modern web stack backed by robust data retrieval services.

## 2. System Architecture
StockMind is a Go monolith serving a React SPA, backed by three data stores and two external
data providers:
*   **Frontend (React/Vite)**: Provides the user interface, including the AI chat interface, price boards, and market research dashboards.
*   **Backend (Go/Chi)**: A single binary. Handles REST requests, acts as an SSE (Server-Sent Events) server for the streaming chat, persists sessions in PostgreSQL, and runs the agentic LLM loop together with its in-process tools and the hybrid RAG pipeline. The provider is chosen at startup via `LLM_PROVIDER` (`openai` | `anthropic` | `openrouter`); all three default to OpenRouter's base URL, so the setting mainly picks which SDK and streaming shape is used.
*   **Data stores**: PostgreSQL (users, sessions, messages, documents, watchlist, research reports), Qdrant (dense + sparse vectors for RAG), MinIO (uploaded document files).
*   **External data providers**: VietCap for Vietnamese market data — prices, financial reports, company insight — and Tavily for web search and news. Both are called directly over HTTP from Go; there is no separate tool server.

## 3. Core Capabilities
The features of StockMind are divided into the conversational AI capabilities and the visual dashboard tools.

### 3.1 Conversational AI Flow
When a user interacts with the Chatbot:
1.  The user's text is sent to the Go backend (`POST /v1/chat`).
2.  The backend resolves or creates the session, persists the user turn, and runs the turn against the configured LLM provider.
3.  When the model needs data it emits a tool call, which the backend executes **in-process**: `get_stock_price`, `get_report`, `get_news`, `fundamental_analysis`, `retrieve_knowledge`, plus any tool bridged from an external MCP server. The result is appended to the history and the loop repeats until the model stops calling tools.
4.  Deltas stream back to the React UI in real time as SSE events: `start`, `thinking`, `text`, `tool_call`, `tool_result`, `done`, `error`.

The `max_mode` flag on the request picks the flow: off runs the single agentic loop above, on
runs the planned multi-agent pipeline (`internal/orchestration` over `internal/agents`). Both
share the same session, persistence and SSE relay path.

### 3.2 Market Dashboard
The dashboard uses traditional REST endpoints to retrieve JSON: `/v1/stock/*` for the price board, watchlist, research reports and fundamental analysis, and `/v1/news` for recent news.

## 4. Directory Structure

The repository holds both tiers: a Go monolith (`cmd/` + `internal/`) and the React SPA (`frontend/`), with the SQL-first data layer in `schema/`. The tables below are the map - which folder owns what, and what filenames look like inside it.

**Authoritative detail lives in [Project Structure](./project-structure.md)**: full trees, per-folder naming conventions, Go/TypeScript identifier rules, and the places where current code deviates from convention. Update that file first; this section is the summary.

### 4.1 Top level

| Path | Contains | Naming |
| --- | --- | --- |
| `cmd/main.go` | CLI entry point (urfave/cli/v3). Only one subcommand: `server` | — |
| `internal/` | All Go code, unexportable outside the module | see 4.2 |
| `schema/migrations/` | Goose DDL migrations, applied at startup | `NNNN_<description>.sql` |
| `schema/queries/` | sqlc query definitions — the source of `internal/database` | `<table>.sql`, one file per table |
| `schema/knowledge_base/` | RAG seed documents (PDF/XLSX); symbol/industry CSVs under `data/` | — |
| `frontend/` | React 19 + Vite + TypeScript SPA; builds to `frontend/dist`, served by the Go server | see 4.3 |
| `docs/` | Project documentation | `kebab-case.md` |
| `Makefile`, `Dockerfile`, `docker-compose.yml`, `sqlc.yaml`, `.air.toml` | Build, infra and codegen config | — |

### 4.2 Backend packages - `internal/`

| Package | Responsibility | File naming |
| --- | --- | --- |
| `server/` | chi HTTP layer, SSE streaming, serves the SPA | `<resource>.handler.go`; `server.go` / `routes.go` for wiring |
| `llm/` | Agentic loop (`service.go`), history compression (`summarizer.go`) | — |
| `llm/models/` | One adapter per LLM provider, selected by `LLM_PROVIDER` | `<provider>.go` |
| `llm/prompts/` | Prompt templates as text files, loaded at runtime | `<task>_prompt.txt`, `agent_<name>.txt` |
| `llm/tools/` | Tool contract + registry merging native and bridged MCP tools | `base_tool.go`, `tool_registry.go` |
| `llm/tools/implementations/` | Execution logic for each native tool | `<tool_name>.handler.go` — filename matches the tool name the LLM sees |
| `agents/` | Specialist agents for the planned multi-agent pipeline (max mode) | `<name>.agent.go`, one agent per file |
| `orchestration/` | Executes a plan's steps in order, emits progress events. Depends on `agents/`, never the reverse | — |
| `knowledge/` | Hybrid RAG: parse → chunk → embed → BM25 → Qdrant, fused with RRF | one file per pipeline stage |
| `mcp/` | MCP **client** manager for external MCP servers (not a server) | `client.go`, `manager.go` |
| `database/` | sqlc-generated data access + hand-written migration runner and JSONB types | `<table>.sql.go` generated — never hand-edit |
| `service/` | External services and background work: `tavily/` web search, `worker/` elastic pools | `<job>.go` defines `<Job>Job` + `<Job>Worker` |
| `storage/` | MinIO object store for uploaded document files | `minio.go` |
| `common/` | Env-only config loading, logging, JSON/SSE response helpers, domain constants | — |

### 4.3 Frontend - `frontend/src/`

| Path | Contains | Naming |
| --- | --- | --- |
| `router.tsx` | All routes in one place (`createBrowserRouter`) | — |
| `api/` | One module per backend resource, over a shared axios instance in `index.ts` | `<resource>.ts`, snake_case, plural for collections |
| `types/` | Shared entity and API-event types | `<entity>.ts`, snake_case, singular |
| `hooks/` | Custom React hooks, one per file | `use<Thing>.ts` |
| `lib/` | Pure logic with no React dependency | `<what-it-does>.ts`, kebab-case |
| `pages/` | Route-level components | `<Name>Page.tsx`, default export |
| `components/ui/` | shadcn primitives, generated by the CLI; no domain knowledge | `<name>.tsx`, kebab-case |
| `components/layout/` | Page shell and navigation, used with `<Outlet />` | PascalCase, named export |
| `components/containers/` | Feature components that hold state and call the API | `<Name>.tsx`, default export |
| `components/*.tsx` | Domain components reused across features | PascalCase, named export |

API payload fields stay `snake_case` on the TypeScript side to match the Go JSON tags - no camelCasing at the API boundary.
