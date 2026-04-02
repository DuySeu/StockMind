# Phase 02: Document Parser - Context

**Gathered:** 2026-04-01
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the core logic for extracting raw text from uploaded files (PDF, DOCX, Markdown, TXT) and validating the quality of the extracted text before it proceeds to chunking and embedding in later phases.

**Included in Scope:**
- `internal/rag/parser.go` with interface `Parser` and multiple implementations.
- PDF text extraction using a specialized Go library (e.g., `ledongthuc/pdf`).
- DOCX text extraction using a specialized Go library (e.g., `nguyenthenguyen/docx`).
- Markdown to plaintext conversion that preserves table structure and link text for context.
- Language-aware validation: Ensuring extracted text is valid Vietnamese or English and contains at least 100 meaningful characters.

**Out of Scope:**
- File upload handling (this is in Phase 4).
- Chunking strategies (this is in Phase 3).
- Storing vectors in Qdrant (this is in Phase 3).
- UI components for document management (this is in Phase 6).
</domain>

<decisions>
## Implementation Decisions

### PDF & DOCX Libraries
- **D-01:** We will use specialized 3rd-party libraries for better reliability:
  - PDF: `github.com/ledongthuc/pdf` (or `rsc.io/pdf`) instead of `pdfcpu`.
  - DOCX: `github.com/nguyenthenguyen/docx` instead of manual XML parsing.

### Validation & Quality Check
- **D-02:** Use language detection (e.g., `github.com/pemistahl/lingua-go` or `github.com/abadojack/whatlanggo`) to ensure text is primarily VI or EN.
- **D-03:** A minimum of 100 meaningful characters is required after normalizing whitespace and stripping boilerplate. If a document fails this, it should be marked as `failed` with a clear message ("Văn bản trích xuất không đủ ý nghĩa hoặc không đúng định dạng").

### Text Pre-processing
- **D-04:** Special Markdown handling:
  - Tables: Preserve structure (e.g., rows separated by newlines, columns formatted clearly) instead of just stripping `|` and `-`.
  - Links: Preserve the "Link Text" and optionally the URL if it provides useful context (e.g., `[Link Text](URL)` → `Link Text (URL)`).

### Error Handling
- **D-05:** Implement a fallback mechanism or explicit errors for scanned (image-only) PDFs: "Không thể trích xuất văn bản từ file này (có thể là file scan hoặc bị khóa)".
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project & Roadmap
- `.planning/ROADMAP.md` §Phase 2 — Core deliverables and success criteria.
- `.planning/REQUIREMENTS.md` §PROC-01, PROC-02 — Document processing requirements.
- `.planning/PROJECT.md` — Overall vision and tech stack constraints.

### Suggested Libraries
- `github.com/ledongthuc/pdf` — PDF text extraction (Confirmed).
- `github.com/nguyenthenguyen/docx` — DOCX document handling.
- `github.com/abadojack/whatlanggo` — Lightweight language detection for Go.
- `github.com/yuin/goldmark` — Markdown parsing for Go.
</canonical_refs>
