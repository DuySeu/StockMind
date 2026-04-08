package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestRetrieveKnowledgeHandler_NilStore(t *testing.T) {
	handler := NewRetrieveKnowledgeHandler(nil, nil)
	
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"query": "What is P/E?",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.IsError {
		t.Errorf("expected error result due to nil store/embedder")
	}
}
