package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"stockmind/internal/rag"
)

// NewRetrieveKnowledgeHandler creates an MCP tool handler for querying the RAG knowledge base.
func NewRetrieveKnowledgeHandler(store rag.Store, embedder rag.Embedder) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if store == nil || embedder == nil {
			return mcp.NewToolResultError("RAG storage or embedder is not initialized"), nil
		}

		// Embed query
		vector, err := embedder.EmbedQuery(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to embed query: %v", err)), nil
		}

		// Search
		results, err := store.Search(ctx, vector, 5, 0.70)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search knowledge base: %v", err)), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("No relevant information found in knowledge base."), nil
		}

		// Format output
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
		for i, res := range results {
			sb.WriteString(fmt.Sprintf("--- Source %d | Doc: %s, Chunk: %d ---\n", i+1, res.DocID, res.ChunkIndex))
			sb.WriteString(res.Text)
			sb.WriteString("\n\n")
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}
