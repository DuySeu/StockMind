# Phase 02: Document Parser - Research

## Objective
Identify and validate libraries and patterns for extracting high-quality raw text from PDF, DOCX, Markdown, and TXT files in Go. Ensure validation logic supports language detection and context preservation.

## Library Evaluation

### 1. PDF Extraction
- **Chosen:** `github.com/ledongthuc/pdf`
- **Why:** Lightweight, stable, and provides a direct `GetPlainText()` reader. Forked from `rsc.io/pdf` with improvements for Vietnamese encoding and better text extraction results.
- **Decision:** Use this for all PDF parsing to ensure reliable text extraction from financial reports.
- **Risk:** Might struggle with very complex multi-column layouts, but usually sufficient for financial reports.

### 2. DOCX Extraction
- **Candidate:** `github.com/nguyenthenguyen/docx`
- **Why:** Recommended in initial brainstorms for balanced reading/editing.
- **Investigation:** While often used for search/replace, its `Editable` model allows reading the XML content.
- **Alternative:** `github.com/beevik/docx` or manual unzip + `encoding/xml` for `word/document.xml`. Given the 3rd party requirement, `nguyenthenguyen/docx` or `beevik/docx` are better choices.

### 3. Markdown Optimization
- **Candidate:** `github.com/yuin/goldmark` (Parser) + Custom Plaintext Renderer.
- **Why:** `go-strip-markdown` is too aggressive (strips tables entirely). Using `goldmark` allows us to traverse the AST and render tables as structured text (e.g., pipe-separated or tab-separated lines) and preserve link text.
- **Implementation:** Create a custom `renderer.NodeRenderer` for Goldmark.

### 4. Language Detection
- **Candidate:** `github.com/abadojack/whatlanggo`
- **Why:** Extremely fast, zero dependencies, supports both Vietnamese and English.
- **Usage:** Run detection on the first ~500-1000 characters of extracted text to validate the document type.

## Technical Patterns

### Parser Interface
```go
type Parser interface {
    Parse(r io.Reader) (string, error)
}
```
Implementations: `PDFParser`, `DOCXParser`, `MDParser`, `TXTParser`.

### Text Quality Validation
1. **Clean:** Remove non-printable characters.
2. **Detect Language:** Must be VI or EN with confidence > 0.8.
3. **Min Length:** > 100 non-whitespace characters.
4. **Boilerplate Detection:** (Optional) Strip common headers/footers if patterns are obvious.

## Integration & Constraints
- Libraries must be added to `go.mod`.
- Ensure files are read as UTF-8. Use `golang.org/x/net/html/charset` for TXT file detection if needed.

## Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Scanned PDF (no text layer) | `ledongthuc/pdf` returns empty string; detect this and return explicit "Scanned PDF" error. |
| Large files memory spike | Use streaming readers where possible; apply 10MB limit (already in budget). |
| Complex tables in DOCX | Use a library approach that preserves table row boundaries. |
