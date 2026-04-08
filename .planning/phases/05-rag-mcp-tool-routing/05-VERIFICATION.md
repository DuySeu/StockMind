---
status: human_needed
score: 4/5
---

# Phase 05: RAG MCP Tool & Intent Routing

## Goal Achievement
**Goal:** Integrate the RAG knowledge base into the AI Assistant via an MCP tool, enabling the LLM to automatically retrieve relevant information based on user intent.
**Status:** Requires human verification to ensure the LLM understands the tool description and uses it properly. automated criteria met.

## Requirement Verification

| ID | Description | Status | Evidence/Notes |
|---|---|---|---|
| RAG-01 | MCP tool `retrieve_knowledge` registered | ✅ Passed | Registered in `internal/mcp/service.go`. |
| RAG-02 | Tool embeds query using same model | ✅ Passed | `EmbedQuery` added to `Embedder` interface. |
| RAG-03 | Search Qdrant top-5 with threshold ≥ 0.70 | ✅ Passed | `Search` method implemented in `QdrantStore`. |
| RAG-04 | "No results" case gracefully handled | ✅ Passed | Returns "No relevant information found..." if empty. |
| RAG-05 | Precise tool description | ✅ Passed | Description explicitly mentions to not use for live prices. |
| RAG-06 | LLM orchestrates retrieval based on intent | ⏳ Pending | Needs human verification. |

## Automated Checks
- [x] Search API interface compiled and tests pass (`go test ./internal/rag`)
- [x] RAG Tool initialized without panic if no store (`go test ./internal/mcp`)
- [x] Main cmd builds cleanly with MCP injection (`go build ./cmd/main.go`)

## Human Verification Required

1. **Test Intent Routing (Static Knowledge):**
   - **expected:** Ask "Tỷ lệ P/E là gì?" -> LLM uses `retrieve_knowledge`.
2. **Test Intent Routing (Live Data):**
   - **expected:** Ask "Giá VNM bao nhiêu?" -> LLM does *not* use `retrieve_knowledge`, uses `get_stock_price` instead.
3. **Test Intent Routing (Combination):**
   - **expected:** Ask "Phân tích HPG và cho tôi biết giá hiện tại." -> LLM uses both tools correctly.
