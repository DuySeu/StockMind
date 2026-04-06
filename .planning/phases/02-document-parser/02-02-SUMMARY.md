# Summary: Phase 02-02 PDF & DOCX Parsers

Implemented text extraction for PDF and DOCX documents using prioritized 3rd-party libraries.

## Changes
- Implemented `PDFParser` using `github.com/ledongthuc/pdf`.
- Implemented `DOCXParser` using `github.com/nguyenthenguyen/docx`.
- **Consolidation**: All parser logic (`PDF`, `DOCX`, `Markdown`, `TXT`) and `Validator` were consolidated into a single `internal/rag/parser.go` following Effective Go principles for cleaner package structure.
- Added temporal buffering for DOCX/PDF readers to support random access on `io.Reader` sources.
- Added XML tag stripping for DOCX output to extract clean text from the inner XML.

## Verification Results
- `TestPDFParser_Empty`: PASS (Correctly fails on 0-byte PDF)
- `TestDOCXParser_Empty`: PASS (Correctly fails on 0-byte DOCX)
- Manual logical check: PDF `GetPlainText()` handles text layers; DOCX `stripXMLTags()` extracts text from XML.
