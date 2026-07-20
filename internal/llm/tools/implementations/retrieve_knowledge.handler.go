package implementations

import (
	"context"
	"fmt"
	"strings"

	kb "stockmind/internal/knowledge"
)

type RetrieveKnowledgeInput struct {
	Query string `json:"query" jsonschema:"Query related to financial knowledge or concepts"`
}

func HandleRetrieveKnowledge(ctx context.Context, retriever kb.Retriever, input RetrieveKnowledgeInput) (any, error) {
	if strings.TrimSpace(input.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	results, err := retriever.Search(ctx, input.Query, kb.SearchHybrid, 5)
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
