# Phase 5 Context: RAG MCP Tool & Intent Routing

**Phase:** 5
**Goal:** Integrate the RAG knowledge base into the AI Assistant via an MCP tool, enabling the LLM to automatically retrieve relevant information based on user intent.

## Scope

- Implement semantic search in the `Store` and `Embedder` layers.
- Create an MCP tool `retrieve_knowledge` with a precise description.
- Register the tool in the MCP engine.
- Verify that the LLM uses the tool when appropriate (financial definitions, concepts) and avoids it for other queries (real-time data, calculations).

## Requirements Mapping

| Requirement | Description | Status |
|-------------|-------------|--------|
| **RAG-01** | MCP tool `retrieve_knowledge` registered in MCP engine | ⬜ Pending |
| **RAG-02** | Tool embeds query using `nvidia/llama-nemotron-embed-vl-1b-v2:free` | ⬜ Pending |
| **RAG-03** | Search Qdrant top-5 with threshold ≥ 0.70 | ⬜ Pending |
| **RAG-04** | Handle "No results" case gracefully | ⬜ Pending |
| **RAG-05** | Precise tool description for intent-based routing | ⬜ Pending |
| **RAG-06** | LLM orchestrates retrieval based on conversation context | ⬜ Pending |

## Technical Decisions

- **Tool Name:** `retrieve_knowledge`
- **Arguments:** `query (string)`
- **Embedding Model:** `nvidia/llama-nemotron-embed-vl-1b-v2:free` via OpenRouter (2048 dimensions).
- **Search Logic:**
    1. Embed query string.
    2. Search Qdrant collection `stockmind_knowledge`.
    3. Limit results to top 5.
    4. Filter by score ≥ 0.70 (cosine similarity).
- **Response Format:** Markdown-formatted list of chunks with source references (document name, chunk index).

## Dependencies

- **Phase 3 (Chunking, Embedding & Storage):** Must provide the `internal/rag` package with `Embedder` and `Store` interfaces/implementations.
- **Phase 1 (Infrastructure):** Qdrant must be running and the collection must exist.
- **MCP Engine:** Existing framework in `internal/mcp/service.go`.

## Pitfalls to Avoid

1. **Model Mismatch:** Ensure the query embedding model exactly matches the document embedding model (2048 dimensions).
2. **Ambiguous Tool Description:** If the description is too broad, the LLM might call it for everything, increasing latency unnecessarily.
3. **Large Response Payloads:** Concatenating too many chunks might hit token limits or degrade response quality. Limit to top 5 and ensure chunks are not excessively large.
