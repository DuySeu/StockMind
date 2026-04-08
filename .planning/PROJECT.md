# StockMind — AI Financial Assistant

StockMind là một trợ lý tài chính AI thông minh cho thị trường chứng khoán Việt Nam, kết hợp dữ liệu thị trường thời gian thực với khả năng truy xuất kiến thức chuyên sâu (RAG).

## Current Milestone: v3.0 (Advanced RAG & Multimodal)

**Goal:** Enhance the RAG pipeline with multimodal capabilities and advanced retrieval techniques.

**Target features:**
- **Multimodal Support**: Integration of vision-capable models for chart analysis.
- **Hybrid Search**: Advanced lexical + semantic retrieval.
- **Citations**: Source attribution in RAG responses.

---

## Future Goals (v4.0+)

1. **User Isolation**: Multi-tenant knowledge isolation.
2. **Agentic Workflows**: Multi-step reasoning chains for complex financial analysis.

---

## Shipped Milestones

### v2.0 Shipped (Agent & MCP Refactor)
> Completed: 2026-05-20

Pivot the core agent engine to be robust, SDK-compliant, and fully capable of orchestrating MCP tools with standard streaming.

- **Anthropic SDK Migration**: Official SDK integration across all agent providers.
- **MCP Tool Orchestration**: Reliable detection and execution of external tools.
- **Streaming Standardization**: Go-idiomatic streaming of both content and tool blocks.
- **Bug Remediation**: Hardening against current nil-pointer and serialization errors.

---

## Future Goals (v2.1+)

1. **Citations**: Source attribution in RAG responses.
2. **User Isolation**: Multi-tenant knowledge isolation.
3. **Hybrid Search**: Advanced lexical + semantic retrieval.

---

## Shipped Milestones

### v1.5 Shipped (RAG Knowledge Base)
> Completed: 2026-04-08

Hệ thống RAG đã được triển khai đầy đủ, cho phép chatbot trả lời các câu hỏi về kiến thức tài chính dựa trên tài liệu người dùng cung cấp.

- **Async Processing Pipeline**: Tự động parse (PDF, DOCX, MD, TXT), chunk (recursive/semantic), và embed tài liệu.
- **Vector Storage**: Tích hợp Qdrant Vector DB để lưu trữ và tìm kiếm vector 2048 chiều.
- **AI Integration**: MCP tool `retrieve_knowledge` cho phép LLM tự động truy xuất kiến thức khi cần.
- **Frontend Management**: Giao diện người dùng toàn diện để upload, theo dõi trạng thái và quản lý tài liệu.

---

## Core Value

Chatbot StockMind trả lời câu hỏi tài chính chính xác hơn nhờ tìm kiếm trong knowledge base nội bộ, đồng thời vẫn giữ khả năng gọi agent tools cho các tác vụ khác — tất cả được điều phối tự động dựa trên intent.

---

## Technical Context

- **Backend**: Go 1.25.1, `chi` (routing), `pgx` (db), `sqlc` (codegen), `goose` (migrations).
- **AI/LLM**: OpenRouter API (`nvidia/llama-nemotron-embed-vl-1b-v2:free`), MCP Framework.
- **Vector DB**: Qdrant (Self-hosted via Docker).
- **Frontend**: React 19, Vite, TailwindCSS 4, Radix UI.

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
*Last Updated: 2026-04-08 (Milestone v2.0 Inception)*
