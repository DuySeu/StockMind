package agents

import (
	"fmt"
	"sort"
	"strings"
)

// Registry holds the built agent roster. Read-only after construction, so it is
// safe to share across requests.
type Registry struct {
	byName map[string]Agent
	names  []string // sorted, for deterministic planner prompts and error messages
}

// NewRegistry builds the default roster. Adding an agent means adding one file in
// this package and one line here.
func NewRegistry(d Deps) *Registry {
	return NewRegistryFrom(
		NewMarketDataAgent(d),
		NewNewsAgent(d),
		NewKnowledgeAgent(d),
		NewFundamentalAgent(d),
		NewSynthesizerAgent(d),
	)
}

// NewRegistryFrom builds a registry from an explicit agent list. Tests use this to
// install fakes.
func NewRegistryFrom(list ...Agent) *Registry {
	r := &Registry{
		byName: make(map[string]Agent, len(list)),
		names:  make([]string, 0, len(list)),
	}
	for _, a := range list {
		r.byName[a.Name()] = a
		r.names = append(r.names, a.Name())
	}
	sort.Strings(r.names)
	return r
}

// Get returns the named agent.
func (r *Registry) Get(name string) (Agent, error) {
	a, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("unknown agent %q (available: %s)", name, strings.Join(r.names, ", "))
	}
	return a, nil
}

// Names returns every registered agent name, sorted.
func (r *Registry) Names() []string {
	out := make([]string, len(r.names))
	copy(out, r.names)
	return out
}

// Roster returns the planner-facing summary of every agent, sorted by name.
func (r *Registry) Roster() []AgentInfo {
	out := make([]AgentInfo, 0, len(r.names))
	for _, name := range r.names {
		a := r.byName[name]
		out = append(out, AgentInfo{
			Name:        a.Name(),
			Description: a.Description(),
			Tools:       a.Tools(),
		})
	}
	return out
}

// RenderRoster formats a roster for inclusion in the planning prompt.
func RenderRoster(roster []AgentInfo) string {
	var sb strings.Builder
	for _, a := range roster {
		fmt.Fprintf(&sb, "- %s: %s\n", a.Name, a.Description)
		if len(a.Tools) == 0 {
			sb.WriteString("  tools: none (reasons over the outputs of earlier steps)\n")
			continue
		}
		fmt.Fprintf(&sb, "  tools: %s\n", strings.Join(a.Tools, ", "))
	}
	return sb.String()
}
