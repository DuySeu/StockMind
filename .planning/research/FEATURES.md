# Features Research — StockMind RAG Feature

**Domain:** RAG document management + financial knowledge assistant
**Researched:** 2026-04-01
**Confidence:** HIGH

## Feature Categories

### Table Stakes (Must Have)

| Feature | Complexity | Notes |
|---------|-----------|-------|
| Multi-format document upload (PDF, DOCX, MD, TXT) | Medium | Need format-specific parsers |
| Async processing pipeline with status tracking | Medium | Upload → pending → processing → ready/failed |
| Vector similarity search | Low | Qdrant handles this |
| Intent-based routing (RAG vs. agent tools) | Medium | LLM decides via tool definition + system prompt |
| Document list + delete in UI | Low | CRUD operations on metadata |
| Processing status indicator in UI | Low | Poll or WebSocket push |
| Chunking strategy selection per upload | Medium | 3-4 strategies exposed as options |
| Metadata stored in PostgreSQL | Low | Document name, type, status, chunk count, timestamps |

### Differentiators (Nice to Have — Later)

| Feature | Complexity | Future Phase |
|---------|-----------|-------------|
| Source citation in answers | Medium | Add chunk reference tracking now, surface in UI later |
| Per-user knowledge bases | High | Requires auth + collection namespacing |
| Re-indexing / update documents | Medium | Delete + re-upload workaround for now |
| Full-text search within docs | Medium | Hybrid search Qdrant BM25 |
| Document chunking preview | High | Show chunks before indexing |
| Automatic language detection | Low | nomic-embed handles multilingual naturally |
| Financial entity extraction | High | NER tagging (stocks, company names, ratios) |

### Anti-Features (Deliberately Not Building)

| Feature | Why Not |
|---------|---------|
| OCR for scanned PDFs | Adds heavy dependency (tesseract), complex setup for 10MB limit |
| Image/chart extraction from PDFs | Complex, low ROI for text-based financial docs |
| Real-time collaborative editing | Not a document editor |
| Version control for documents | Delete + re-upload is sufficient |
| Team/role-based permissions | Deferred; single-user context for now |

## UX Patterns for Document Management

### Upload Flow (Industry Standard)
```
User selects file → 
  Frontend validates (type, size ≤10MB) →
  POST /api/documents →
  Backend saves file + creates DB record (status: pending) →
  Returns {document_id, status: "pending"} immediately →
  Background goroutine: parse → chunk → embed → update status →
  Frontend polls GET /api/documents/{id} or shows spinner
```

**Key UX principle:** Never block the user during processing. Show optimistic UI with a processing state.

### Document List (Industry Standard)
- Show: filename, upload date, status badge (Processing / Ready / Failed), file type icon, chunk count
- Actions: delete (with confirmation), re-upload if failed
- Status polling: every 2-3 seconds while any doc is "processing"
- Empty state: "Upload your first document to enhance AI answers"

### Chunking Strategy Selection (UX Decision)
Present as a simple dropdown on the upload form:

| Option | Label | Best For |
|--------|-------|---------|
| `recursive` | Smart Split (Recommended) | Most documents |
| `fixed` | Fixed Size | Structured reports, uniform text |
| `paragraph` | By Paragraph | Articles, research papers |
| `semantic` | By Topic | Mixed-topic documents |

## RAG Chat Integration Patterns

### Intent Routing via Tool Description
The most reliable approach: make the `retrieve_knowledge` MCP tool description very precise so the LLM routes correctly:

```
Tool: retrieve_knowledge
Description: Search the internal financial knowledge base for information about 
Vietnamese stock market terminology, investment concepts, financial definitions, 
regulatory information, and company/sector analysis. Use this tool when the user 
asks about definitions, concepts, or background knowledge that may be in uploaded 
documents. Do NOT use this for real-time stock prices, live market data, or 
calculations.
```

This clear boundary prevents the LLM from conflating RAG with live data tools.

### Query Expansion (Best Practice)
Before sending query to Qdrant, have the LLM expand the query:
- User: "P/E là gì?" → Expanded: "price to earnings ratio P/E definition Vietnam stock market"
- Improves recall for Vietnamese/English mixed documents

---
*Features research for: StockMind RAG Feature*
*Researched: 2026-04-01*
