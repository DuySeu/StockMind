# StockMind — AI Financial Assistant

StockMind là một trợ lý tài chính AI thông minh cho thị trường chứng khoán Việt Nam, kết hợp dữ liệu thị trường thời gian thực với khả năng truy xuất kiến thức chuyên sâu (RAG).

## Current State: v1.5 Shipped (RAG Knowledge Base)
> Completed: 2026-04-08

Hệ thống RAG đã được triển khai đầy đủ, cho phép chatbot trả lời các câu hỏi về kiến thức tài chính dựa trên tài liệu người dùng cung cấp.

### Key Features Shipped:
- **Async Processing Pipeline**: Tự động parse (PDF, DOCX, MD, TXT), chunk (recursive/semantic), và embed tài liệu.
- **Vector Storage**: Tích hợp Qdrant Vector DB để lưu trữ và tìm kiếm vector 2048 chiều.
- **AI Integration**: MCP tool `retrieve_knowledge` cho phép LLM tự động truy xuất kiến thức khi cần.
- **Frontend Management**: Giao diện người dùng toàn diện để upload, theo dõi trạng thái và quản lý tài liệu.

---

## Core Value

Chatbot StockMind trả lời câu hỏi tài chính chính xác hơn nhờ tìm kiếm trong knowledge base nội bộ, đồng thời vẫn giữ khả năng gọi agent tools cho các tác vụ khác — tất cả được điều phối tự động dựa trên intent.

---

## Next Milestone Goals (v2.0)

Tập trung vào tính minh bạch (Citation), trải nghiệm người dùng nâng cao và cá nhân hóa:
1. **Citations**: Trích dẫn nguồn tài liệu trong câu trả lời của AI.
2. **User Isolation**: Mỗi người dùng có một không gian kiến thức riêng biệt.
3. **Hybrid Search**: Kết hợp tìm kiếm từ khóa và vector để tăng độ chính xác.

---

## Technical Context

- **Backend**: Go 1.25.1, `chi` (routing), `pgx` (db), `sqlc` (codegen), `goose` (migrations).
- **AI/LLM**: OpenRouter API (`nvidia/llama-nemotron-embed-vl-1b-v2:free`), MCP Framework.
- **Vector DB**: Qdrant (Self-hosted via Docker).
- **Frontend**: React 19, Vite, TailwindCSS 4, Radix UI.

---
*Last Updated: 2026-04-08 (Milestone v1.5 Completion)*
