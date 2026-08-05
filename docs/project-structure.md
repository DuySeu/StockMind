# Project Structure

Tài liệu mô tả cấu trúc thư mục của StockMind và quy ước đặt tên trong từng folder. Chia làm ba phần: **root-level structure**, **backend** (Go), **frontend** (React SPA).

Mọi đường dẫn tính từ repo root. Tài liệu này phản ánh code thực tế; nếu code và tài liệu lệch nhau, code thắng — và hãy cập nhật file này.

---

## 1. Root-level structure

StockMind là một **Go monolith** phục vụ một **React SPA**. Backend là một binary
duy nhất; frontend build ra static files và được chính Go server serve.

```
StockMind/
├── cmd/                    # Entry point của binary
│   └── main.go             # CLI (urfave/cli/v3) — chỉ một subcommand: `server`
├── internal/               # Toàn bộ Go code (không export ra ngoài module)
├── schema/                 # SQL-first data layer + seed data
│   ├── migrations/         # Goose migrations (DDL)
│   │   └── NNNN_<mô_tả>.sql    # Số tăng dần (0001_init.sql); không sửa file đã chạy
│   ├── queries/            # sqlc query definitions (nguồn của internal/database)
│   │   └── <table>.sql         # snake_case, một file / bảng, khớp <table>.sql.go sinh ra
│   └── knowledge_base/     # PDF/XLSX seed cho RAG và mapping báo cáo tài chính
│       └── data/               # CSV danh mục mã / ngành
├── frontend/               # React 19 + Vite + TypeScript SPA
├── docs/                   # Tài liệu dự án (file này nằm ở đây)
│   └── <tên>.md            # kebab-case (ngoại lệ đang có: UI-PLAN.md)
├── .cursor/ .kiro/ .claude/# Cấu hình agent/AI tooling — không phải application code
├── Makefile                # run / build / test / itest / watch / docker-*
├── Dockerfile              # Multi-stage build (Go binary + frontend dist)
├── docker-compose.yml      # PostgreSQL 17 + Qdrant + MinIO cho local dev
├── sqlc.yaml               # Cấu hình codegen: schema/queries → internal/database
├── .air.toml               # Live reload (make watch)
├── .env / .env.example     # Config env-only; `.env` bị gitignore
├── go.mod / go.sum         # Module name: `stockmind`
├── CLAUDE.md               # Hướng dẫn cho AI agent làm việc trong repo
└── main                    # Binary đã build — gitignored, không commit
```

---

## 2. Backend — `internal/`

```
internal/
├── agents/                 # Specialist agents cho multi-agent pipeline (max mode)
│   ├── base.go             # Agent interface, Input/Output, Emit, baseAgent dùng chung
│   ├── plan.go             # Plan contract (kiểu dữ liệu kế hoạch + validate)
│   ├── planner.go          # LLM planner: goal + roster → Plan
│   ├── registry.go         # Roster mặc định; thêm agent = thêm 1 dòng ở đây
│   └── <name>.agent.go     # Một agent / file, dựng trên base.go — news, market_data, knowledge, fundamental, synthesizer
├── common/                 # Tiện ích dùng chung, không phụ thuộc package nào khác
│   ├── configs.go          # LoadConfig — đọc os.Getenv, không đọc file config
│   ├── const.go            # Hằng số VietCap / domain
│   ├── gzip.go             # GZIPCompression — giải nén body gzip từ upstream
│   ├── logging.go          # Cấu hình slog
│   └── response.go         # Helper trả JSON / SSE
├── database/               # Data layer (sqlc)
│   ├── <table>.sql.go      # SINH TỰ ĐỘNG, một file / bảng — khớp schema/queries/<table>.sql
│   ├── models.go           # SINH TỰ ĐỘNG — struct theo bảng
│   ├── db.go               # SINH TỰ ĐỘNG — Queries, DBTX
│   ├── migration.go        # MigrateDB — chạy goose lúc startup (viết tay)
│   └── types.go            # Custom type cho cột JSONB (viết tay)
├── knowledge/              # Hybrid RAG pipeline — một file / bước trong pipeline
│   ├── knowledge_base.go   # kb.New() — dựng store + embedder + BM25
│   ├── parser.go           # PDF / DOCX / MD / TXT → text
│   ├── chunker.go          # Text → chunks
│   ├── embedder.go         # Dense vector qua EMBED_MODEL
│   ├── bm25.go             # Sparse vector
│   ├── store.go            # Qdrant upsert / search
│   ├── retriever.go        # Fusion dense + sparse bằng RRF
│   └── ingest.go           # IngestPipeline — điều phối các bước trên
├── llm/                    # Agentic loop + provider + tools + prompts
│   ├── service.go          # LLMService — DB-agnostic, caller tự persist
│   ├── summarizer.go       # Nén history dài để tiết kiệm token
│   ├── models/             # Provider adapter
│   │   ├── llm_factory.go  # Chọn provider theo LLM_PROVIDER
│   │   ├── model_router.go # Route model theo tác vụ
│   │   └── <provider>.go   # Một file / provider — anthropic.go, openai.go, openrouter.go
│   ├── prompts/
│   │   ├── loader.go       # Load + render template
│   │   └── templates/      # Prompt là text file, không hardcode chuỗi trong Go
│   │       ├── <task>_prompt.txt   # Prompt theo tác vụ: system_, plan_, research_, summarization_, metrics_
│   │       ├── agent_<name>.txt    # Khớp Name() của <name>.agent.go
│   │       └── fundamental_analysis.txt  # ⚠ lệch quy ước: thiếu hậu tố `_prompt`
│   └── tools/
│       ├── base_tool.go        # Tool, Manager, NewTool[In] — schema từ struct tag
│       ├── tool_registry.go    # RegisterTools + BridgeMCPTools (wiring)
│       ├── <name>_test.go      # Test cạnh code nó test (hiện chỉ có subset_test.go)
│       └── implementations/    # Logic thực thi từng tool
│           └── <tool_name>.handler.go  # Tên file TRÙNG tên tool LLM nhìn thấy
├── mcp/                    # MCP *client* manager (không còn là MCP server)
│   ├── client.go           # Spawn + nói chuyện với 1 external MCP server
│   └── manager.go          # Quản lý nhiều server, evict + reconnect 1 lần khi lỗi
├── orchestration/          # Chạy plan nhiều bước theo đúng thứ tự
│   ├── orchestrator.go     # Budget/timeout, thực thi step, phát event
│   └── event.go            # Kiểu event phát ra cho UI
├── server/                 # HTTP layer (chi)
│   ├── server.go           # struct Server + dependency
│   ├── routes.go           # RegisterRoutes + spaHandler (serve frontend/dist)
│   ├── <resource>.handler.go   # Handler nhóm theo resource — chat, session, user,
│   │                           #   document, news, stock, agent_flow, researcher, websocket
│   └── fundamental_analysis.go # ⚠ lệch quy ước: thiếu `.handler.go`
├── service/                # Service bên ngoài + background work
│   ├── service.go          # struct Services — nơi wire mọi service
│   ├── tavily/             # Web search client — một file / nhóm endpoint
│   │   ├── client.go       # struct Client + NewClient
│   │   ├── research.go     # SubmitResearch / PollResearch
│   │   └── search_news.go  # SearchWeb
│   └── worker/             # Worker pool đàn hồi
│       ├── worker_pool.go  # Pool[T] generic dùng chung
│       └── <job>.go        # Một file / loại job — document.go, research.go
│                           #   mỗi file định nghĩa <Job>Job + <Job>Worker
└── storage/
    └── minio.go            # ObjectStore interface + MinIO implementation
```

---

## 3. Frontend — `frontend/`

```
frontend/
├── index.html              # Vite entry HTML
├── vite.config.ts          # Alias `@` → ./src, proxy /api → localhost:8080
├── components.json         # Cấu hình shadcn/ui (style new-york, base neutral)
├── eslint.config.js
├── tsconfig.json           # Project references → app + node
├── tsconfig.app.json / tsconfig.node.json
├── package.json            # scripts: dev / build / lint / preview
├── public/                 # Asset serve nguyên trạng (stockmind.png)
├── dist/                   # Build output — gitignored, Go server serve folder này
└── src/
    ├── main.tsx            # Bootstrap React + router
    ├── router.tsx          # createBrowserRouter — toàn bộ route ở một chỗ
    ├── index.css           # Tailwind v4 + CSS variable theme
    ├── vite-env.d.ts
    ├── assets/             # Asset được bundler xử lý
    ├── api/                # HTTP client theo resource
    │   ├── index.ts        # axios instance + interceptor (baseURL http://localhost:8080/v1)
    │   └── <resource>.ts   # snake_case, số NHIỀU khi là collection — chat, sessions, document, news, stock, agent_flows
    ├── types/              # Type dùng chung giữa các module
    │   └── <entity>.ts     # snake_case, số ÍT — message, document, stock, research, agent_flow
    ├── hooks/              # Custom React hook — một hook / file
    │   ├── use<Thing>.ts   # camelCase cho hook tự viết — useDocumentPolling.ts
    │   └── use-mobile.ts   # ⚠ kebab-case vì shadcn sinh ra
    ├── lib/                # Logic thuần, không phụ thuộc React
    │   ├── utils.ts        # cn() — bắt buộc phải có cho shadcn
    │   └── <việc-nó-làm>.ts    # kebab-case — pdf-export.ts, validation.ts, stock.ts
    ├── pages/              # Component gắn với một route; export default
    │   ├── <Name>Page.tsx  # PascalCase + hậu tố Page — Home, Login, Error, Pending, Document, Setting, MarketResearcher, ResearchResult
    │   ├── Chatbot.tsx     # ⚠ lệch quy ước: file thiếu hậu tố Page (export ChatbotPage)
    │   └── WatchList.tsx   # ⚠ lệch quy ước: file thiếu hậu tố Page (export WatchListPage)
    └── components/
        ├── ui/             # shadcn primitives — sinh bằng CLI, hạn chế sửa tay
        │   └── <name>.tsx  # kebab-case do CLI đặt — button, dialog, table, sidebar, dropdown-menu, scroll-area, …
        ├── layout/         # Khung trang, dùng với <Outlet />; export named
        │   ├── MainLayout.tsx
        │   └── Navbar.tsx
        ├── containers/     # Component theo feature: có state, gọi API; export default
        │   └── <Name>.tsx  # PascalCase, không hậu tố — ChatInput, MessageList, SideBar, Header, ResearchReport
        └── <Name>.tsx      # Component domain dùng lại ở nhiều feature; export named DocumentListTable, DocumentUploadForm, StatusBadge
```
