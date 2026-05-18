package tools

import (
	"context"
	"fmt"
	"strings"

	kb "stockmind/internal/knowledge_base"
)

func init() {
	Register(
		ToolSchema{
			Name:        "retrieve_knowledge",
			Description: "Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Query related to financial knowledge or concepts",
					},
				},
				"required": []string{"query"},
			},
		},
		func(deps InternalToolDeps) HandlerFunc {
			return func(ctx context.Context, args map[string]any) (map[string]any, error) {
				query, _ := args["query"].(string)
				if strings.TrimSpace(query) == "" {
					return nil, fmt.Errorf("query is required")
				}

				results, err := deps.Retriever.Search(ctx, query, kb.SearchHybrid, 5)
				if err != nil {
					return nil, fmt.Errorf("knowledge base search failed: %w", err)
				}

				if len(results) == 0 {
					return map[string]any{"content": "No relevant information found in knowledge base."}, nil
				}

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
				for i, res := range results {
					sb.WriteString(fmt.Sprintf("--- Source %d | Doc: %s, Chunk: %d ---\n", i+1, res.DocID, res.ChunkIndex))
					sb.WriteString(res.Text)
					sb.WriteString("\n\n")
				}

				return map[string]any{"content": sb.String()}, nil
			}
		},
	)
}
