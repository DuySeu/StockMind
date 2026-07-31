package agents

// NewNewsAgent builds the recent-news and events specialist.
func NewNewsAgent(d Deps) Agent {
	return &LLMAgent{
		name: "news",
		desc: "Searches for recent news, announcements and market events about a ticker, sector or the " +
			"market as a whole. Use whenever the goal depends on what has happened lately. " +
			"Does not cover prices, financial statements or company profiles.",
		promptTpl: "agent_news.txt",
		toolNames: []string{"get_news"},
		deps:      d,
	}
}
