package mcp

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/service/tavily"

	"github.com/mark3labs/mcp-go/mcp"
)

func GetNews(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client := tavily.NewClient()

	result, err := client.SearchWeb(ctx, query, common.NEWS_DOMAINS)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	fmt.Println(result)
	return mcp.NewToolResultStructuredOnly(result), nil
}
