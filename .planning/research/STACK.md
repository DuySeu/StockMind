# Stack Research — StockMind RAG Feature

**Domain:** RAG pipeline + Vector DB + Go backend
**Researched:** 2026-04-01
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/qdrant/go-client` | v1.17.x | Qdrant vector DB client | Official Go client, gRPC-based (faster than REST), active maintenance |
| `github.com/sashabaranov/go-openai` | existing | Embedding calls to OpenRouter | Already in codebase, supports base URL override → OpenRouter endpoint |
| `golang.org/x/sync/errgroup` | latest | Async pipeline goroutine management | Cleaner than raw WaitGroup, auto-cancels context on first error |
| Qdrant Docker image | `qdrant/qdrant:latest` | Self-hosted vector store | Add to existing docker-compose.yml |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pdfcpu/pdfcpu` | v0.9.x | PDF text extraction | Open-source (Apache 2.0), text extraction from PDFs |
| `archive/zip` (stdlib) | stdlib | DOCX parsing (OpenXML = ZIP + XML) | DOCX = zip of XML files, no extra dep needed |
| `encoding/xml` (stdlib) | stdlib | Parse word/document.xml from DOCX | Native Go, works with archive/zip |
| `github.com/russross/blackfriday/v2` | v2 | Markdown → plaintext | For .md files, strip formatting before chunking |
| `bufio` + `unicode` (stdlib) | stdlib | TXT processing | Plain text needs minimal processing |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Qdrant Web UI | Inspect collections, debug vectors | Available at `localhost:6333/dashboard` when running Docker |
| `go get github.com/qdrant/go-client` | Add Qdrant client | One-time setup |

## Embedding Model Selection

**OpenRouter Free Embedding Models (verified via `/api/v1/embeddings/models`):**

| Model | Dimensions | Context | Notes |
|-------|-----------|---------|-------|
| `nomic-ai/nomic-embed-text-v1.5` | 768 | 8192 tokens | **Recommended** — strong multilingual, good for Vietnamese |
| `jina-ai/jina-embeddings-v2-base-en` | 768 | 8192 tokens | English-only, less suitable for Vietnamese financial terms |
| `thenlper/gte-large` | 1024 | 512 tokens | High quality but short context window |

**Recommendation: `nomic-ai/nomic-embed-text-v1.5`**
- Best multilingual support (important for Vietnamese + English mixed financial docs)
- 8192 token context fits most document chunks comfortably
- Free tier on OpenRouter
- 768 dimensions → reasonable Qdrant storage

## Qdrant Configuration

```yaml
# docker-compose.yml addition
qdrant:
  image: qdrant/qdrant:latest
  ports:
    - "6333:6333"   # REST API + Web UI
    - "6334:6334"   # gRPC (used by Go client)
  volumes:
    - qdrant_data:/qdrant/storage
```

**Collection settings:**
- Distance: `Cosine` (aligned with nomic-embed-text training)
- Vector size: 768
- `on_disk: false` for ≤50 docs scale (RAM is fine)

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `pdfcpu` | `unipdf` (UniDoc) | Only if need complex PDF manipulation/generation; commercial license required |
| OpenXML stdlib parsing | `docxgo` | When need to write/edit DOCX files (we only read) |
| `nomic-embed-text-v1.5` | `text-embedding-ada-002` | When willing to pay; ada-002 is paid |
| Qdrant self-hosted | Qdrant Cloud | When deployment scale requires managed infra |
| `errgroup` | raw goroutines | Only for trivial single-goroutine cases |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `unipdf` | Commercial license, overkill for text extraction | `pdfcpu` (Apache 2.0) |
| Weaviate / Pinecone | Adds new infra paradigm, complex setup | Qdrant (simpler, already chosen) |
| LangChain Go ports | Immature, poorly maintained Go ports | Implement pipeline directly in Go |
| Word-level chunking | Too granular, poor semantic coherence | Sentence/paragraph-level chunking |

---
*Stack research for: StockMind RAG Feature*
*Researched: 2026-04-01*
