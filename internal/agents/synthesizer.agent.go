package agents

// NewSynthesizerAgent builds the final agent, which merges the findings of earlier
// steps into the single answer the user reads.
//
// It has no tools: toolNames is non-nil but empty, which tools.Manager.Subset
// reads as "exactly zero tools" (nil would mean "all of them").
func NewSynthesizerAgent(d Deps) Agent {
	return &LLMAgent{
		name: SynthesizerAgentName,
		desc: "Merges the findings of earlier steps into one coherent final answer. Has no tools and " +
			"cannot fetch anything — it only reasons over what earlier agents produced. " +
			"Must always be the last step, and its use_output_of should name every step whose " +
			"findings belong in the answer.",
		promptTpl: "agent_synthesizer.txt",
		toolNames: []string{},
		deps:      d,
	}
}
