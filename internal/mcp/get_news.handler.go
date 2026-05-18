package mcp

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/service/tavily"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetNewsInput struct {
	Query string `json:"query" jsonschema:"Query related to stock news"`
}

func GetNews(ctx context.Context, req *mcp.CallToolRequest, input GetNewsInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return nil, nil, fmt.Errorf("query is required")
	}

	client := tavily.NewClient()
	result, err := client.SearchWeb(ctx, input.Query, common.NEWS_DOMAINS)
	if err != nil {
		return nil, nil, err
	}

	return nil, result, nil
}
