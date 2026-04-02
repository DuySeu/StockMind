# Roadmap: StockMind RAG Knowledge Base

**Milestone:** RAG Feature
**Created:** 2026-04-01
**Requirements:** 34 v1 requirements
**Phases:** 6
**Coverage:** 34/34 ✓

---

## Phase 1: Infrastructure & Database

**Goal:** Qdrant chạy trong Docker, bảng `documents` tồn tại trong PostgreSQL, backend kết nối được với Qdrant khi startup.

**Requirements:** INFRA-01, INFRA-02, INFRA-03, INFRA-04

**Deliverables:**
- Qdrant service thêm vào `docker-compose.yml` với healthcheck và volume
- Goose migration: tạo bảng `documents`
- sqlc queries cho documents: insert, get, list, update_status, delete
- Qdrant client init trong Go với retry + collection auto-create
- Unit-tested collection init logic

**Success Criteria:**
1. `docker compose up` khởi động Qdrant healthy tại `localhost:6333/dashboard`
2. `make migrate` chạy thành công, bảng `documents` tồn tại với đúng schema
3. Backend startup log: "Qdrant collection ready: stockmind_knowledge"
4. Backend tự retry khi Qdrant chưa sẵn sàng (kiểm tra bằng khởi động backend trước Qdrant)

---

## Phase 2: Document Parser

**Goal:** Backend có thể parse text từ file PDF, DOCX, Markdown, TXT và validate chất lượng output.

**Requirements:** PROC-01, PROC-02

**Deliverables:**
- `internal/rag/parser.go` với interface `Parser` và 4 implementations
- PDF parser dùng `pdfcpu`: extract text từ text-based PDFs
- DOCX parser dùng stdlib: unzip → parse `word/document.xml`
- MD parser: strip markdown formatting → plaintext
- TXT parser: UTF-8 read with encoding detection
- Text quality validator: reject nếu < 100 meaningful chars

**Success Criteria:**
1. Parse PDF thông thường (VD: báo cáo tài chính) → text sạch, có nội dung
2. Parse DOCX file → text sections đầy đủ
3. Parse `.md` file → plaintext không có Markdown symbols
4. File PDF scan (ảnh) → trả về error rõ ràng: "Không thể trích xuất văn bản từ file này"
5. Test coverage ≥ 80% cho parser package

---

## Phase 3: Chunking, Embedding & Qdrant Storage

**Goal:** Text đã parse được chunk theo strategy, embed qua OpenRouter, lưu vào Qdrant thành công.

**Requirements:** PROC-03, PROC-04, PROC-05, PROC-06, PROC-07

**Deliverables:**
- `internal/rag/chunker.go` — interface `Chunker` + strategy enum
- `chunker_fixed.go` — fixed-size 512 tokens với overlap
- `chunker_recursive.go` — recursive character splitting (default)
- `chunker_paragraph.go` — paragraph boundary detection
- `chunker_semantic.go` — embedding-based semantic boundary detection
- `internal/rag/embedder.go` — OpenRouter embedding với go-openai, batch=20
- `internal/rag/store.go` — Qdrant upsert với payload {doc_id, chunk_index, text}
- Model consistency check: store model name in collection metadata

**Success Criteria:**
1. Sample PDF → 512-token chunks với 51-token overlap (recursive strategy)
2. OpenRouter embedding call thành công với `nomic-ai/nomic-embed-text-v1.5` → vector 768-dim
3. Batch upsert 100 chunks → Qdrant collection có 100 points với payload đúng
4. Semantic search trả về kết quả liên quan cho sample query
5. Model mismatch detection: nếu collection dùng model khác → error rõ khi startup

---

## Phase 4: Async Worker & Document Service + REST API

**Goal:** Upload document qua API → xử lý async → status tracking hoạt động đầy đủ.

**Requirements:** UPLOAD-01, UPLOAD-03, UPLOAD-04, UPLOAD-05, PROC-08, PROC-09, DOC-01, DOC-02, DOC-03

**Deliverables:**
- `internal/rag/worker.go` — goroutine pool (2 workers), buffered channel (cap=10), context-aware shutdown
- `internal/service/document.go` — DocumentService: upload, list, get, delete
- `internal/server/document_handler.go` — HTTP handlers:
  - `POST /api/documents` (multipart, 10MB limit) → 202 Accepted
  - `GET /api/documents` → list all
  - `GET /api/documents/:id` → single doc detail
  - `DELETE /api/documents/:id` → delete doc + Qdrant vectors
- File size validation server-side (10MB max) → 413 Request Entity Too Large
- Temp file cleanup on success và failure
- Graceful shutdown: drain in-flight jobs before exit

**Success Criteria:**
1. Upload 5MB PDF → HTTP 202 trong < 500ms (không đợi processing)
2. `GET /api/documents` ngay sau upload → document xuất hiện với `status: pending`
3. Sau ~30 giây → `status: ready`, `chunk_count > 0`
4. Upload file lỗi scan → `status: failed`, `error_msg` có nội dung mô tả
5. `DELETE /api/documents/:id` → document xóa khỏi DB, vectors xóa khỏi Qdrant
6. Upload file 11MB → 413 error
7. Server SIGTERM → in-flight job hoàn thành trước khi process exit

---

## Phase 5: RAG MCP Tool & Intent Routing

**Goal:** LLM tự quyết định dùng `retrieve_knowledge` tool khi user hỏi về khái niệm/định nghĩa tài chính.

**Requirements:** RAG-01, RAG-02, RAG-03, RAG-04, RAG-05, RAG-06

**Deliverables:**
- `internal/mcp/rag_tool.go` — `retrieve_knowledge` MCP tool đăng ký vào MCP engine
- Tool nhận `query: string`, embed query, search Qdrant top-5 với threshold 0.70
- Format output: concatenated chunks với ngăn cách rõ ràng
- Precision tool description phân biệt rõ: knowledge base vs. live market data
- Integration test: chat session với documents đã indexed

**Success Criteria:**
1. "Tỷ lệ P/E là gì?" → LLM gọi `retrieve_knowledge` → trả về chunks liên quan từ tài liệu đã upload
2. "Giá cổ phiếu VNM hôm nay?" → LLM KHÔNG gọi `retrieve_knowledge`, gọi tool khác hoặc trả lời trực tiếp
3. Query không có kết quả liên quan → chatbot trả lời "Không tìm thấy thông tin trong knowledge base"
4. Score threshold: mock query với cosine < 0.70 → không trả về kết quả
5. Response latency thêm ≤ 2 giây so với chat không dùng RAG

---

## Phase 6: Frontend Document Management UI

**Goal:** User có thể upload, xem, và xóa tài liệu qua giao diện trực quan trên React frontend.

**Requirements:** UPLOAD-02, UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07

**Deliverables:**
- Route `/documents` với layout phù hợp sidebar/nav hiện tại
- `DocumentUploadForm` component: file picker, strategy dropdown, upload button, progress indicator
- `DocumentList` component: table/grid với status badges, chunk count, delete action
- Status badge component: Processing (spinner), Ready (green), Failed (red + tooltip error)
- Auto-polling hook: `useDocumentPolling` — active khi có doc "processing"
- Delete confirmation dialog (Radix Dialog)
- Frontend file validation: type check + 10MB limit trước khi POST
- API client functions cho document endpoints
- Empty state illustration + copy

**Success Criteria:**
1. Upload PDF từ `/documents` → badge "Processing" xuất hiện ngay; sau xử lý xong → badge "Ready"
2. Document "Failed" → tooltip/expand hiển thị error message từ server
3. Delete document → confirmation dialog → document biến mất khỏi list
4. Upload file không đúng định dạng → validation error ngay trên browser, không POST lên server
5. Upload file > 10MB → validation error ngay trên browser
6. Trang `/documents` không có file nào → empty state hiển thị đúng
7. Responsive: giao diện đúng trên mobile và desktop

---

## Milestone Summary

| Phase | Name | Requirements | Est. Complexity |
|-------|------|-------------|-----------------|
| 1 | 3/3 | Complete    | 2026-04-01 |
| 2 | Document Parser | PROC-01~02 | Medium |
| 3 | Chunking, Embedding & Storage | Complete    | 2026-04-02 |
| 4 | Async Worker & REST API | Planned | High |
| 5 | RAG MCP Tool & Routing | RAG-01~06 | Medium |
| 6 | Frontend UI | UPLOAD-02, UI-01~07 | Medium |

**Build order rationale:** Infrastructure first (1) → parsing foundation (2) → processing pipeline (3) → orchestration layer (4) → AI integration (5) → user-facing layer (6).

---
*Roadmap created: 2026-04-01*
*Last updated: 2026-04-01 after initial creation*
