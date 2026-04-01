# Requirements: StockMind RAG Knowledge Base

**Defined:** 2026-04-01
**Core Value:** Chatbot StockMind trả lời câu hỏi tài chính chính xác hơn nhờ tìm kiếm trong knowledge base nội bộ, đồng thời vẫn giữ khả năng gọi agent tools cho các tác vụ khác — tất cả được điều phối tự động dựa trên intent.

## v1 Requirements

### Infrastructure

- [ ] **INFRA-01**: Qdrant self-hosted chạy qua Docker Compose, accessible từ Go backend (gRPC port 6334)
- [ ] **INFRA-02**: Qdrant collection `stockmind_knowledge` được khởi tạo tự động khi server start (768 dimensions, cosine distance)
- [ ] **INFRA-03**: Backend retry Qdrant connection khi startup với exponential backoff (5 lần, 1s→16s)
- [ ] **INFRA-04**: Goose migration tạo bảng `documents` với các fields: id, name, file_type, size_bytes, status, chunk_count, strategy, error_msg, created_at, updated_at

### Document Upload

- [ ] **UPLOAD-01**: User upload file từ frontend, hỗ trợ định dạng: PDF, DOCX, MD, TXT
- [ ] **UPLOAD-02**: Frontend validate file trước khi upload: đúng định dạng, không quá 10MB
- [ ] **UPLOAD-03**: User chọn chunking strategy khi upload: Smart Split (recursive), Fixed Size, By Paragraph, By Topic (semantic)
- [ ] **UPLOAD-04**: Upload endpoint trả về HTTP 202 Accepted ngay lập tức với `{id, status: "pending"}` — không block chờ processing
- [ ] **UPLOAD-05**: File được lưu tạm thời, record được tạo trong DB với `status = pending`

### Document Processing

- [ ] **PROC-01**: Background worker nhận job và parse text từ file: PDF (pdfcpu), DOCX (stdlib zip+xml), MD (strip formatting), TXT (raw)
- [ ] **PROC-02**: Worker validate chất lượng text sau parsing: nếu < 100 ký tự có nghĩa → mark `failed` với thông báo rõ ràng
- [ ] **PROC-03**: Text được chunk theo strategy đã chọn với params mặc định: 512 tokens, 10% overlap
- [ ] **PROC-04**: Chunks được embed bằng `nomic-ai/nomic-embed-text-v1.5` qua OpenRouter API (batch 20 chunks/call)
- [ ] **PROC-05**: Vectors được upsert lên Qdrant với payload chứa: document_id, chunk_index, text content
- [ ] **PROC-06**: Sau khi xong, DB được update: `status = ready`, `chunk_count = N`; file tạm được xóa
- [ ] **PROC-07**: Nếu bất kỳ bước nào lỗi, `status = failed`, `error_msg` chứa nguyên nhân; file tạm được xóa
- [ ] **PROC-08**: Worker pool có tối đa 2 concurrent jobs; wired vào server shutdown context (graceful stop)
- [ ] **PROC-09**: Job queue là buffered channel capacity 10; nếu đầy, upload nhận 503 Service Unavailable

### Document Management

- [ ] **DOC-01**: `GET /api/documents` trả về danh sách tất cả documents với metadata: id, name, file_type, status, chunk_count, created_at
- [ ] **DOC-02**: `GET /api/documents/:id` trả về chi tiết một document kể cả error_msg nếu failed
- [ ] **DOC-03**: `DELETE /api/documents/:id` xóa document record khỏi DB VÀ xóa tất cả vectors liên quan khỏi Qdrant

### RAG Retrieval (MCP Tool)

- [ ] **RAG-01**: MCP tool `retrieve_knowledge` được đăng ký trong MCP engine, nhận tham số `query: string`
- [ ] **RAG-02**: Tool embed query dùng cùng model (`nomic-ai/nomic-embed-text-v1.5`) để đảm bảo tương thích vector space
- [ ] **RAG-03**: Tool tìm kiếm Qdrant top-5 với score threshold ≥ 0.70 (cosine similarity)
- [ ] **RAG-04**: Nếu không có kết quả vượt threshold, tool trả về "No relevant information found in knowledge base"
- [ ] **RAG-05**: Tool description đủ cụ thể để LLM phân biệt: dùng cho định nghĩa/khái niệm/tài liệu, KHÔNG dùng cho giá realtime/news/tính toán
- [ ] **RAG-06**: LLM tự quyết định intent-based: dùng `retrieve_knowledge` hay các agent tools khác

### Frontend UI

- [ ] **UI-01**: Trang Document Management (`/documents`) có form upload với: file picker, strategy dropdown, nút upload
- [ ] **UI-02**: Sau upload thành công, document xuất hiện ngay trong danh sách với badge "Processing"
- [ ] **UI-03**: Danh sách documents hiển thị: tên file, loại file (icon), ngày upload, status badge (Processing/Ready/Failed), số chunks
- [ ] **UI-04**: Frontend poll `GET /api/documents` mỗi 3 giây khi có document đang ở trạng thái "processing"
- [ ] **UI-05**: Document ở trạng thái "Failed" hiển thị tooltip/expand với lý do lỗi
- [ ] **UI-06**: User có thể xóa document (có confirmation dialog); xóa xong cập nhật danh sách
- [ ] **UI-07**: Empty state khi chưa có document: "Upload tài liệu để tăng cường khả năng trả lời của AI"

## v2 Requirements

### Citation & Transparency

- **CITE-01**: Câu trả lời dựa trên RAG có trích dẫn nguồn: tên tài liệu, số chunk
- **CITE-02**: UI hiển thị "Sources used" collapsible section bên dưới câu trả lời

### Per-User Knowledge Base

- **USER-01**: Mỗi user có collection Qdrant riêng (namespace theo user_id)
- **USER-02**: Upload/delete chỉ ảnh hưởng knowledge base của user đó

### Admin Controls

- **ADMIN-01**: Admin có thể xem toàn bộ documents của tất cả users
- **ADMIN-02**: Admin có thể xóa bất kỳ document nào
- **ADMIN-03**: Giới hạn số documents per user (mặc định 50)

### Quality Improvements

- **QUAL-01**: Query expansion trước khi embed: LLM mở rộng query ngắn thành câu đầy đủ hơn
- **QUAL-02**: Score threshold có thể cấu hình qua env var `RAG_SCORE_THRESHOLD`
- **QUAL-03**: Top-K có thể cấu hình qua env var `RAG_TOP_K`
- **QUAL-04**: Re-indexing document mà không cần delete + re-upload

## Out of Scope

| Feature | Reason |
|---------|--------|
| OCR cho scanned PDFs | Cần thêm Tesseract dependency nặng, phức tạp setup |
| Trích xuất ảnh/bảng từ PDF | Low ROI, scope creep |
| Hybrid search (keyword + vector) | Có thể add sau nếu recall rate thấp (v2+) |
| Document versioning/update | Delete + re-upload đủ dùng cho v1 |
| Phân quyền upload/delete | Deferred, tất cả user đều có quyền trong v1 |
| Real-time processing status qua WebSocket | Polling đủ tốt cho 50 docs max |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 1 | Pending |
| INFRA-02 | Phase 1 | Pending |
| INFRA-03 | Phase 1 | Pending |
| INFRA-04 | Phase 1 | Pending |
| UPLOAD-01 | Phase 4 | Pending |
| UPLOAD-02 | Phase 6 | Pending |
| UPLOAD-03 | Phase 4 + Phase 6 | Pending |
| UPLOAD-04 | Phase 4 | Pending |
| UPLOAD-05 | Phase 4 | Pending |
| PROC-01 | Phase 2 | Pending |
| PROC-02 | Phase 2 | Pending |
| PROC-03 | Phase 3 | Pending |
| PROC-04 | Phase 3 | Pending |
| PROC-05 | Phase 3 | Pending |
| PROC-06 | Phase 3 | Pending |
| PROC-07 | Phase 3 | Pending |
| PROC-08 | Phase 4 | Pending |
| PROC-09 | Phase 4 | Pending |
| DOC-01 | Phase 4 | Pending |
| DOC-02 | Phase 4 | Pending |
| DOC-03 | Phase 4 | Pending |
| RAG-01 | Phase 5 | Pending |
| RAG-02 | Phase 5 | Pending |
| RAG-03 | Phase 5 | Pending |
| RAG-04 | Phase 5 | Pending |
| RAG-05 | Phase 5 | Pending |
| RAG-06 | Phase 5 | Pending |
| UI-01 | Phase 6 | Pending |
| UI-02 | Phase 6 | Pending |
| UI-03 | Phase 6 | Pending |
| UI-04 | Phase 6 | Pending |
| UI-05 | Phase 6 | Pending |
| UI-06 | Phase 6 | Pending |
| UI-07 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 34 total
- Mapped to phases: 34
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-01*
*Last updated: 2026-04-01 after initial definition*
