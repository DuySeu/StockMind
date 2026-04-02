# Phase 02: Document Parser - Discussion Log

**Date:** 2026-04-01

## Q&A Session

### 1. PDF Extraction Library
- **Question:** Roadmap đề xuất `pdfcpu`, nhưng thư viện này thiên về chỉnh sửa. Có nên cân nhắc các thư viện chuyên trích xuất text như `github.com/ledongthuc/pdf` không?
- **Response:** "có"
- **Decision:** Sử dụng thư viện chuyên dụng (`ledongthuc/pdf`) để tối ưu việc trúng xuất text.

### 2. DOCX Extraction Strategy
- **Question:** Tự parse XML hay dùng thư viện bên thứ 3?
- **Response:** "xử dụng thư viện bên thứ 3"
- **Decision:** Sử dụng thư viện `nguyenthenguyen/docx` để đảm bảo độ ổn định khi xử lý cấu trúc XML nội bộ của file Word.

### 3. Text Quality Validation
- **Question:** Định nghĩa văn bản "có ý nghĩa" như thế nào?
- **Response:** "kiểm tra ngôn ngữ"
- **Decision:** Sử dụng giải thuật detect ngôn ngữ (ưu tiên VI, EN) và kiểm tra mật độ ký tự hợp lệ thay vì chỉ đếm độ dài chuỗi.

### 4. Markdown Processing
- **Question:** Chỉ xóa format Markdown hay cần xử lý đặc biệt (bảng, link)?
- **Response:** "cần xử lý đặc biệt để giữ lại thông tin ngữ canh"
- **Decision:** Giữ lại cấu trúc bảng (table) và text trong link để AI không bị mất ngữ cảnh so với văn bản gốc.

## Status Mapping
- `PROC-01`: Fully addressed with library choices.
- `PROC-02`: Fully addressed with language validation logic.
