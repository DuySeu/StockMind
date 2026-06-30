package tools

import (
	"context"

	kb "stockmind/internal/knowledge_base"
	"stockmind/internal/service"
)

// StructuredCompleter performs a non-streaming LLM completion whose JSON output
// is unmarshalled into result (a pointer). It is provider-agnostic and supplied
// by the core LLM package, avoiding an import cycle (core imports tools).
type StructuredCompleter func(ctx context.Context, prompt string, result any) error

// RegisterTools creates all tool definitions with their dependencies via closures.
func RegisterTools(retriever kb.Retriever, services *service.Services) []*Tool {
	toolList := []*Tool{
		NewTool("retrieve_knowledge",
			"Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news.",
			func(ctx context.Context, input RetrieveKnowledgeInput) (any, error) {
				return HandleRetrieveKnowledge(ctx, retriever, input)
			},
		),

		NewTool("get_stock_price",
			"Get OHLC stock price data for a Vietnamese stock symbol from VietCap.",
			HandleGetStockPrice,
		),

		NewTool("get_report",
			"Get quarterly or yearly financial report for a stock.",
			HandleGetReport,
		),

		NewTool("get_news",
			"Get stock news for a query via web search.",
			func(ctx context.Context, input GetNewsInput) (any, error) {
				return HandleGetNews(ctx, *services.Tavily, input)
			},
		),

		NewTool("fundamental_analysis",
			"Produce a qualitative fundamental analysis of a Vietnamese stock: company overview, shareholder structure, ecosystem, business activity, economic moat, outlook, risks, and macro context. Grounds factual fields on VietCap company data and synthesizes the analytical narrative.",
			func(ctx context.Context, input FundamentalAnalysisInput) (any, error) {
				return HandleFundamentalAnalysis(ctx, input)
			},
		),
	}

	return toolList
}
