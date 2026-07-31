package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func stubTool(name string) *Tool {
	return &Tool{
		Name:        name,
		Description: name,
		Schema:      map[string]any{"type": "object"},
		Execute: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		},
	}
}

func names(ts []*Tool) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Name)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSubsetNilVersusEmpty pins the distinction the agent pipeline depends on:
// nil means "no restriction" (the chat path's zero value) while a non-nil empty
// slice means "no tools at all" (a synthesizer agent). Collapsing the two would
// silently hand every tool to an agent that declared none.
func TestSubsetNilVersusEmpty(t *testing.T) {
	m := NewManager([]*Tool{stubTool("a"), stubTool("b")})

	if got := names(m.Subset(nil)); !equal(got, []string{"a", "b"}) {
		t.Errorf("Subset(nil) = %v, want every tool", got)
	}
	if got := m.Subset([]string{}); len(got) != 0 {
		t.Errorf("Subset([]) = %v, want no tools", names(got))
	}
}

func TestSubsetSelectsNamedToolsInOrder(t *testing.T) {
	m := NewManager([]*Tool{stubTool("a"), stubTool("b"), stubTool("c")})

	got := names(m.Subset([]string{"c", "a"}))
	if !equal(got, []string{"c", "a"}) {
		t.Errorf("Subset([c a]) = %v, want [c a] in the order given", got)
	}
}

// TestSubsetSkipsUnknownNames covers a legitimately absent tool — e.g. a bridged
// MCP server that failed to start — which must not take the whole agent down.
func TestSubsetSkipsUnknownNames(t *testing.T) {
	m := NewManager([]*Tool{stubTool("a")})

	got := names(m.Subset([]string{"a", "ghost"}))
	if !equal(got, []string{"a"}) {
		t.Errorf("Subset([a ghost]) = %v, want [a]", got)
	}
}

func TestSubsetOnEmptyManager(t *testing.T) {
	m := NewManager(nil)

	if got := m.Subset(nil); len(got) != 0 {
		t.Errorf("Subset(nil) on an empty manager = %v, want none", names(got))
	}
	if got := m.Subset([]string{"a"}); len(got) != 0 {
		t.Errorf("Subset([a]) on an empty manager = %v, want none", names(got))
	}
}
