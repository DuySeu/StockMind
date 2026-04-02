---
phase: 5
plan: 2
name: mcp-tool-integration
wave: 1
depends_on: [search-api-extension]
requirements: [RAG-01, RAG-05, RAG-06]
files_modified: [internal/mcp/service.go, internal/mcp/rag_tool.go]
autonomous: true
---

# Plan 02: MCP Tool Integration

Registering the `retrieve_knowledge` tool with a specific description to enable intent-based retrieval by the LLM.

## Tasks

<task id="5-02-01" requirements="RAG-01, RAG-05">
<action>
Create `internal/mcp/rag_tool.go`.
Define `RetrieveKnowledgeHandler` using `rag.Store` and `rag.Embedder`.
The handler should:
1.  Receive query from MCP request.
2.  Embed query.
3.  Search store with `topK=5, threshold=0.70`.
4.  If zero results, return "No relevant information found in knowledge base."
5.  Format result as a clear list of chunks with source doc reference.
</action>
<read_first>
- internal/mcp/service.go
- internal/rag/store.go
- internal/rag/embedder.go
</read_first>
<acceptance_criteria>
- `internal/mcp/rag_tool.go` defines the tool handler.
- Description is very precise about NOT using it for real-time prices or news.
</acceptance_criteria>
</task>

<task id="5-02-02" requirements="RAG-01">
<action>
Register the new tool in `internal/mcp/service.go`.
Use `s.AddTool(mcp.NewTool("retrieve_knowledge", ...), RetrieveKnowledgeHandler)`.
Description: "Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news."
</action>
<read_first>
- internal/mcp/service.go
- internal/mcp/rag_tool.go
</read_first>
<acceptance_criteria>
- Tool is registered and visible to the LLM agent.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Description ensures correct intent routing by the model.
- Threshold of 0.70 prevents returning irrelevant noise.
- Top-5 keeps context window manageable.
</must_haves>

<automated>
- `go test -v ./internal/mcp -run TestRetrieveKnowledge`
- Manual check using MCP Inspector or similar to see tool availability.
</automated>
