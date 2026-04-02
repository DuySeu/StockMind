# Phase 5 Research: RAG Tool Design

## Embedding Model Parameters
- **Model:** `nvidia/llama-nemotron-embed-vl-1b-v2:free` via OpenRouter.
- **Dimensions:** 2048.
- **Task:** Symetrical embedding (query and document chunks use the same model).

## Search Strategy
- **Metric:** Cosine Similarity (default in Qdrant for this model).
- **Threshold (0.70):** Chosen to ensure high relevance but allow for semantically similar results. 0.70 is standard for many similarity-based retrieval tasks. If the recall is too low, we might need a lower threshold (e.g., 0.60).
- **Top-K (5):** Balances context and token usage. 5 chunks of ~512 characters/tokens are well within LLM context windows (e.g., NEMOTRON NANO, etc.).

## Intent Routing
- The LLM will distinguish between "live" and "static" knowledge via the Tool Description.
- **Live Tool Description (e.g., `get_stock_price`):** "Latest, real-time price."
- **Knowledge Tool Description (`retrieve_knowledge`):** "Concepts, definitions, detailed background, historical contexts from internal docs."

## MCP Go Framework
- Using `github.com/mark3labs/mcp-go`.
- Tools are added via `s.AddTool`.
- Each tool needs a name, description, parameters schema, and a handler function.
