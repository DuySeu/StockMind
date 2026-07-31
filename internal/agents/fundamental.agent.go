package agents

// NewFundamentalAgent builds the company profile and fundamentals specialist.
func NewFundamentalAgent(d Deps) Agent {
	return &LLMAgent{
		name: "fundamental",
		desc: "Retrieves and analyses a company's fundamentals: business model, industry and competitive " +
			"position, shareholder structure, group relationships and key ratios. " +
			"Use for questions about what a company is and how sound its business is. " +
			"Does not cover recent news or intraday prices.",
		promptTpl: "agent_fundamental.txt",
		toolNames: []string{"fundamental_analysis"},
		deps:      d,
	}
}
