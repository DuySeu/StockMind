package agents

// NewKnowledgeAgent builds the internal-document retrieval specialist.
func NewKnowledgeAgent(d Deps) Agent {
	return &LLMAgent{
		name: "knowledge",
		desc: "Searches the internal knowledge base of ingested documents for financial concepts, " +
			"definitions, methodology and in-house analysis. Use for conceptual or explanatory " +
			"questions, and for anything that should be grounded in the organisation's own documents. " +
			"Does not fetch live market data.",
		promptTpl: "agent_knowledge.txt",
		toolNames: []string{"retrieve_knowledge"},
		deps:      d,
	}
}
