# StockMind

StockMind is an AI-powered financial assistant tailored for the Vietnamese stock market. It streamlines the investment research process by providing intuitive access to financial data, intelligent analysis, and real-time market insights through a conversational AI interface.

---

## Table of Contents

1. [Features](#features)
2. [Tech Stack](#tech-stack)
3. [Architecture](#architecture)
4. [Running Locally](#running-locally)
5. [Project Structure](#project-structure)
6. [Documentation](#documentation)

---

## Features

### Core Capabilities

- **AI Chat Assistant**: Conversational interface powered by LLMs (via OpenRouter) with tool-calling capabilities for real-time financial data access
- **Financial Data Access**: Retrieve official financial statements for any listed Vietnamese company directly through chat
- **Fundamental Analysis**: Automated Piotroski F-Score and Altman Z-Score evaluations via MCP tools
- **Market Research**: Generate deep-dive automated research reports combining web research (Tavily) with LLM analysis
- **Watchlist & Price Board**: Track stocks with real-time price data from VietCap
- **Document Knowledge Base**: Upload financial documents (PDF, DOCX, MD, TXT) for RAG-powered retrieval during chat
- **News Monitoring**: Track financial events, earnings, and regulatory updates

### AI Tools (MCP)

| Tool | Description |
|------|-------------|
| `get_stock_price` | OHLC price data with configurable time frames |
| `piotroski_evaluation` | 9-point Piotroski F-Score analysis |
| `altman_z_score` | Altman Z-Score bankruptcy predictor |
| `get_report` | Quarterly/yearly financial reports |
| `get_news` | Stock news search |
| `retrieve_knowledge` | RAG knowledge base retrieval |

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (chi router, pgx/v5, sqlc) |
| Frontend | React 19 (Vite, TypeScript, Tailwind CSS, shadcn/ui) |
| LLM Integration | OpenRouter (OpenAI + Anthropic SDKs) |
| Tool Protocol | MCP (Model Context Protocol via mcp-go) |
| Database | PostgreSQL 17 |
| Vector Database | Qdrant |
| Web Research | Tavily API |
| Market Data | VietCap Trading API |

---

## Architecture

```
Browser (React SPA)
    ↓ REST + SSE
Chi HTTP Server (Go)
    ↓
Agent Orchestration → LLM Providers (via OpenRouter)
    ↓                       ↓
MCP Tool Server ←→ Financial APIs (VietCap, Tavily)
    ↓
RAG Pipeline → Qdrant (vector storage)
    ↓
PostgreSQL (sessions, reports, documents, users)
```

See [AGENTS.md](./AGENTS.md) for detailed architecture and patterns, or [.agents/summary/architecture.md](.agents/summary/architecture.md) for full diagrams.

---

## Running Locally

### Prerequisites

- [Go](https://go.dev/doc/install) (1.25+)
- [Node.js](https://nodejs.org/en) (v18+)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose

### 1. Configure Environment

Copy the example env file and fill in your API keys:

```bash
cp .env.example .env
```

Required variables:
- `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD` — PostgreSQL connection
- `OPENROUTER_API_KEY` — [OpenRouter](https://openrouter.ai/) API key for LLM access
- `TAVILY_API_KEY` — [Tavily](https://tavily.com/) API key for web research

### 2. Start Infrastructure

```bash
docker-compose up -d
```

This starts PostgreSQL 17 and Qdrant. Database migrations run automatically on app startup.

### 3. Run the Application

```bash
make run
```

This starts the Go backend and the Vite React dev server concurrently.

### 4. Access the Application

- **Frontend**: [http://localhost:5173](http://localhost:5173)
- **Backend API**: [http://localhost:8080](http://localhost:8080)
- **MCP Server** (if HTTP mode): [http://localhost:8081](http://localhost:8081)

### Stopping

```bash
# Stop the app (Ctrl+C on make run)
docker-compose down
```

### Docker Production Build

```bash
make docker-build
# or
docker build -t stockmind .
```

---

## Project Structure

```
cmd/main.go              # CLI entry point (server, mcp subcommands)
internal/
  agent/                 # LLM orchestration, session management, providers
  server/                # HTTP handlers, routing, SSE streaming
  mcp/                   # MCP tool server (financial analysis tools)
  rag/                   # RAG pipeline (parse, chunk, embed, store)
  database/              # sqlc-generated database layer
  service/               # Business logic (documents, Tavily)
  common/                # Constants, utilities
frontend/src/            # React SPA
schema/
  migrations/            # SQL migrations (goose)
  queries/               # sqlc query definitions
  knowledge_base/        # Static financial reference data
```

---

## Documentation

- **[AGENTS.md](./AGENTS.md)** — AI agent context file with architecture, patterns, and navigation guide
- **[.agents/summary/index.md](.agents/summary/index.md)** — Documentation index (start here for deep dives)
- **[.agents/summary/](.agents/summary/)** — Full documentation suite:
  - `architecture.md` — System diagrams and design patterns
  - `components.md` — Per-package component breakdown
  - `interfaces.md` — REST API, MCP tools, SSE events
  - `data_models.md` — Database schema and type definitions
  - `workflows.md` — Chat, research, RAG, and orchestration flows
  - `dependencies.md` — All dependencies with versions
  - `review_notes.md` — Known gaps and improvement suggestions
