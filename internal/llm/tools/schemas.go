package tools

func init() {
	AddTool("retrieve_knowledge",
		"Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news.",
		HandleRetrieveKnowledge,
	)

	AddTool("get_stock_price",
		"Get OHLC stock price data for a Vietnamese stock symbol from VietCap.",
		HandleGetStockPrice,
	)

	AddTool("get_report",
		"Get quarterly or yearly financial report for a stock.",
		HandleGetReport,
	)

	AddTool("get_news",
		"Get stock news for a query via web search.",
		HandleGetNews,
	)
}
