package implementations

import (
	"context"
	"fmt"

	"stockmind/internal/common"
	"stockmind/internal/service/tavily"
)

type GetNewsInput struct {
	Query string `json:"query" jsonschema:"Query related to stock news"`
}

func HandleGetNews(ctx context.Context, tavily tavily.Client, input GetNewsInput) (any, error) {
	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	result, err := tavily.SearchWeb(ctx, input.Query, common.NEWS_DOMAINS)
	if err != nil {
		return nil, err
	}
	return result, nil
}
