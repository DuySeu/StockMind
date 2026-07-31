package agents

import (
	"errors"
	"fmt"
	"strings"
)

// MaxSteps caps how many steps a plan may contain. It exists to bound latency:
// with a 90s per-step timeout this keeps a worst-case run inside the HTTP
// server's 5-minute WriteTimeout.
const MaxSteps = 6

// SynthesizerAgentName is the agent that must produce the final answer. The
// orchestrator appends a step for it when the planner leaves one out.
const SynthesizerAgentName = "synthesizer"

// Plan is the planner's structured output: an ordered list of delegated steps.
type Plan struct {
	Goal  string `json:"goal"`
	Steps []Step `json:"steps"`
}

// Step is one delegation to one agent.
type Step struct {
	ID     string `json:"id"`
	Agent  string `json:"agent"`
	Task   string `json:"task"`
	Reason string `json:"reason"`
	// UseOutputOf names earlier steps whose outputs this step needs. It selects
	// context, it does not schedule: execution is strictly sequential either way.
	// Restricting references to *earlier* steps is what makes sequential execution
	// provably sufficient — no step can ever want an output that doesn't exist yet.
	UseOutputOf []string `json:"use_output_of,omitempty"`
}

// Validate checks the plan against the set of agents that actually exist.
//
// It reports every problem it finds rather than stopping at the first, because
// the errors are fed back to the planner for a repair attempt and one round-trip
// should fix everything at once.
func (p *Plan) Validate(agentNames []string) error {
	known := make(map[string]struct{}, len(agentNames))
	for _, n := range agentNames {
		known[n] = struct{}{}
	}

	var problems []error

	if len(p.Steps) == 0 {
		return errors.New("plan has no steps")
	}
	if len(p.Steps) > MaxSteps {
		problems = append(problems, fmt.Errorf("plan has %d steps, the maximum is %d", len(p.Steps), MaxSteps))
	}

	seen := make(map[string]struct{}, len(p.Steps))
	for i, s := range p.Steps {
		switch {
		case strings.TrimSpace(s.ID) == "":
			problems = append(problems, fmt.Errorf("step %d: id is empty", i+1))
		case contains(seen, s.ID):
			problems = append(problems, fmt.Errorf("step %d: duplicate id %q", i+1, s.ID))
		}

		if strings.TrimSpace(s.Task) == "" {
			problems = append(problems, fmt.Errorf("step %q: task is empty", s.ID))
		}

		if _, ok := known[s.Agent]; !ok {
			problems = append(problems, fmt.Errorf("step %q: unknown agent %q (available: %s)",
				s.ID, s.Agent, strings.Join(agentNames, ", ")))
		}

		for _, ref := range s.UseOutputOf {
			if !contains(seen, ref) {
				// Covers forward references, self-references and typos alike: only
				// IDs already recorded from *previous* iterations are acceptable.
				problems = append(problems, fmt.Errorf(
					"step %q: use_output_of references %q, which is not an earlier step", s.ID, ref))
			}
		}

		// Recorded after checking references, so a step cannot reference itself.
		if s.ID != "" {
			seen[s.ID] = struct{}{}
		}
	}

	return errors.Join(problems...)
}

// AgentNames returns every agent named by the plan, in step order.
func (p *Plan) AgentNames() []string {
	out := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Agent)
	}
	return out
}

func contains(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}
