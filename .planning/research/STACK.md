# Stack Research — StockMind RAG Feature

**Domain:** RAG pipeline + Vector DB + Go backend
**Researched:** 2026-04-01
**Updated:** 2026-04-01 (embedding model confirmed from OpenRouter API)
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

**Selected Model (confirmed free via OpenRouter `/api/v1/embeddings/models` API):**

| Model | Dimensions | Context | Price | Notes |
|-------|-----------|---------|-------|-------|
| `nvidia/llama-nemotron-embed-vl-1b-v2:free` | **2048** | **131,072 tokens** | **$0.00** | ✅ **Selected** — only truly free model, multimodal (text+image) |

**Full OpenRouter embedding catalog (Apr 2026) — other options for reference:**

| Model | Dim | Context | Price | Multilingual |
|-------|-----|---------|-------|-------------|
| `intfloat/multilingual-e5-large` | 1024 | 512 | $0.01/1M | ✅ 90+ langs |
| `baai/bge-m3` | 1024 | 8,192 | $0.01/1M | ✅ 100+ langs |
| `qwen/qwen3-embedding-8b` | — | 32,000 | $0.01/1M | ✅ Multilingual |
| `google/gemini-embedding-001` | — | 20,000 | $0.15/1M | ✅ Top MTEB |

**Why `nvidia/llama-nemotron-embed-vl-1b-v2:free`:**
- Only $0 free model available on OpenRouter as of Apr 2026
- 131,072 token context — effectively no chunk-size limit
- OpenAI-compatible API — drop-in with go-openai SDK + base URL override
- Multimodal (text+image) — future-proof if PDF image extraction added later
- **Upgrade path:** `baai/bge-m3` ($0.01/1M) if Vietnamese recall rate is insufficient

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
- Distance: `Cosine`
- Vector size: **2048** (matches `llama-nemotron-embed-vl-1b-v2` output dimensions)
- `on_disk: false` for ≤50 docs scale (RAM is fine)

> ⚠️ **Critical:** Vector dimension must match embedding model exactly. If model is changed, collection must be deleted and re-created, all documents re-indexed.

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
