# StockMind — RAG Knowledge Base

## What This Is

StockMind là một AI financial assistant cho thị trường chứng khoán Việt Nam. Milestone này tập trung phát triển tính năng **RAG (Retrieval-Augmented Generation)**: cho phép upload tài liệu tài chính (PDF, DOCX, MD, TXT), tự động phân tích và nhúng (embed) vào knowledge base Qdrant, rồi tích hợp retrieval như một MCP tool để chatbot gọi khi cần thiết dựa trên user intent. Người dùng có thể quản lý (xem, xóa) tài liệu đã upload qua giao diện frontend.

## Core Value

Chatbot StockMind trả lời câu hỏi tài chính chính xác hơn nhờ tìm kiếm trong knowledge base nội bộ, đồng thời vẫn giữ khả năng gọi agent tools cho các tác vụ khác — tất cả được điều phối tự động dựa trên intent.

## Requirements

### Validated

<!-- Inferred from existing codebase -->

- ✓ Chat AI với streaming WebSocket — existing
- ✓ MCP tool calling framework (`/internal/mcp`) — existing
- ✓ OpenRouter integration qua go-openai SDK — existing
- ✓ PostgreSQL với sqlc/goose migration — existing
- ✓ React frontend với file upload support (React Hook Form) — existing
- ✓ Docker Compose infrastructure — existing

### Active

<!-- Scope của milestone RAG này -->

- [ ] Upload tài liệu (PDF, DOCX, MD, TXT, tối đa 10MB) từ frontend
- [ ] Async processing pipeline: parse → chunk → embed → lưu vào Qdrant
- [ ] Chunking strategies: fixed-size, semantic, paragraph-based — chọn strategy khi upload
- [ ] Embedding bằng free model trên OpenRouter (qua go-openai SDK, base URL override)
- [ ] Qdrant self-hosted qua Docker (thêm vào docker-compose)
- [ ] RAG MCP tool: `retrieve_knowledge` — agent gọi khi intent phù hợp
- [ ] Intent-based routing: LLM quyết định dùng RAG tool hay agent tools khác
- [ ] Frontend document management: danh sách tài liệu, trạng thái processing, xóa
- [ ] Metadata tài liệu lưu trong PostgreSQL (tên, loại, trạng thái, timestamps)
- [ ] Processing status tracking: pending → processing → ready / failed

### Out of Scope

- Citation/trích dẫn nguồn trong câu trả lời — deferred, sẽ add sau
- Per-user knowledge base — dùng chung global, phân quyền deferred
- Admin-only upload — chưa cần phân quyền
- Re-indexing / update tài liệu đã upload — chỉ hỗ trợ delete + re-upload
- Hybrid search (keyword + vector) — có thể thêm sau nếu cần
- OCR cho ảnh trong PDF — không trong scope

## Context

**Codebase hiện tại:**
- Backend Go 1.25.1 với layered architecture: router → service → agent → mcp → database
- MCP engine (`/internal/mcp`) dùng `github.com/mark3labs/mcp-go` — đây là nơi add `retrieve_knowledge` tool
- OpenRouter API key đã có (`OPENROUTER_API_KEY`) — dùng luôn với base URL override để call embedding endpoint
- Agent layer (`/internal/agent`) hiện quản lý LLM lifecycle + tool orchestration
- PostgreSQL với goose migrations — sẽ thêm bảng `documents` và `document_chunks`
- Frontend có React Hook Form, Radix UI, TailwindCSS — build UI document management trên nền này

**Lý do chọn Qdrant:**
- Native vector database, self-hosted dễ dàng qua Docker
- Go client library chính thức
- Hỗ trợ payload filtering, metadata search tốt

**Lý do dùng OpenRouter cho embedding:**
- API key đã tồn tại trong hệ thống
- Free tier có các model embedding chất lượng (nomic-embed-text, jina-embeddings)
- Tránh thêm API key mới

## Constraints

- **Tech Stack**: Go 1.25.1, existing pgx/sqlc/goose — không thêm ORM mới
- **Compatibility**: Không breaking change REST/WebSocket API contract với frontend hiện tại
- **Storage**: Qdrant self-hosted Docker (không dùng Qdrant Cloud) — thêm vào docker-compose.yml
- **Async Processing**: Upload endpoint trả về ngay sau khi nhận file, processing chạy background goroutine với proper lifecycle management
- **File Size**: Tối đa 10MB per file, tối đa 50 tài liệu total
- **Embedding**: Dùng free model trên OpenRouter — cần chọn model có `/v1/embeddings` endpoint tương thích

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| RAG as MCP Tool (`retrieve_knowledge`) | Tận dụng MCP framework hiện có, LLM tự quyết định khi nào dùng | — Pending |
| Qdrant self-hosted via Docker | Không cần cloud account, phù hợp dev setup hiện tại | — Pending |
| OpenRouter cho embedding (free model) | Tái dùng API key hiện có, zero cost | — Pending |
| Async processing với background goroutine | Upload UX không bị block, cần graceful shutdown | — Pending |
| Metadata trong PostgreSQL, vectors trong Qdrant | Tách biệt relational metadata và vector storage | — Pending |
| Intent routing qua LLM (không classifier riêng) | Đơn giản hơn, LLM đã hiểu context của conversation | — Pending |
| Chunking strategy chọn per-document khi upload | Linh hoạt theo loại tài liệu, nhưng không over-engineer | — Pending |

---

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-01 after initialization*
