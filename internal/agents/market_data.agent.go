package agents

// NewMarketDataAgent builds the quantitative price and financial-statement
// specialist.
func NewMarketDataAgent(d Deps) Agent {
	return &LLMAgent{
		name: "market_data",
		desc: "Fetches quantitative market data for a Vietnamese ticker: current and historical prices, " +
			"and financial statements (balance sheet, income statement, cash flow). " +
			"Use for any question about prices, trading history or reported figures. " +
			"Does not cover news, company profiles or recommendations.",
		promptTpl: "agent_market_data.txt",
		toolNames: []string{"get_stock_price", "get_report"},
		deps:      d,
	}
}
