package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"stockmind/internal/common"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/rag"
	"stockmind/internal/qdrant"

	"github.com/mark3labs/mcp-go/mcp"
)

var (
	ragOnce     sync.Once
	ragStore    qdrant.Store
	ragEmbedder rag.Embedder
	ragInitErr  error
)

func initRAG(ctx context.Context) error {
	ragOnce.Do(func() {
		cfg := common.LoadConfig()
		conn, err := qdrant.InitQdrant(ctx, cfg.Qdrant.Host, cfg.Qdrant.Port)
		if err != nil {
			ragInitErr = fmt.Errorf("failed to connect to Qdrant: %w", err)
			return
		}
		ragStore = qdrant.NewQdrantStore(conn, "")

		orClient, err := core.NewOpenRouterClient(cfg.LLMConfig.OpenRouter)
		if err != nil {
			ragInitErr = fmt.Errorf("failed to init openrouter client: %w", err)
			return
		}
		ragEmbedder = rag.NewOpenRouterEmbedder(orClient, 0)
	})
	return ragInitErr
}

// NewRetrieveKnowledgeHandler is an MCP tool handler for querying the RAG knowledge base.
func NewRetrieveKnowledgeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := initRAG(ctx); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	vector, err := ragEmbedder.EmbedQuery(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to embed query: %v", err)), nil
	}

	results, err := ragStore.Search(ctx, vector, 5, 0.70)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search knowledge base: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No relevant information found in knowledge base."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("--- Source %d | Doc: %s, Chunk: %d ---\n", i+1, res.DocID, res.ChunkIndex))
		sb.WriteString(res.Text)
		sb.WriteString("\n\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}
