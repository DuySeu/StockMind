package tools

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/service/tavily"
)

type GetNewsInput struct {
	Query string `json:"query" jsonschema:"Query related to stock news"`
}

func HandleGetNews(ctx context.Context, _ Deps, input GetNewsInput) (any, error) {
	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	client := tavily.NewClient()
	result, err := client.SearchWeb(ctx, input.Query, common.NEWS_DOMAINS)
	if err != nil {
		return nil, err
	}
	return result, nil
}
