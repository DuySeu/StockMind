// Package agents defines the specialist agents a plan can delegate to, plus the
// plan contract itself. One file per agent; agent.go holds the shared base.
//
// The orchestrator (internal/orchestration) depends on this package, never the
// reverse — which is why Plan lives here rather than there.
package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"
)

// Input is what the orchestrator hands an agent for one plan step.
type Input struct {
	// Task is the planner's instruction for this step.
	Task string
	// Goal is the user's original request, passed through unmodified. Agents need
	// it for grounding the planner cannot supply: most importantly the user's
	// language, since the planner routinely rewrites tasks into English and an
	// agent working from the task alone would answer in the wrong language.
	Goal string
	// Context holds the outputs of the earlier steps this step's UseOutputOf
	// named, keyed by step ID.
	Context map[string]string
}

// Output is one agent's result for one step.
type Output struct {
	Content string
	// Tools records the tool calls the agent made, for UI display and audit.
	Tools []database.Tool
}

// Emit lets an agent stream partial progress out through the orchestrator while
// it works. kind is an orchestration event type ("tool_call", "step_text", …);
// content carries text deltas and data carries structured payloads.
type Emit func(kind string, content string, data any)

// Agent is one specialist in the pipeline.
//
// Run is on the interface rather than Agent being pure metadata so that an agent
// can be entirely custom — a deterministic API call with no LLM, say — without
// the orchestrator needing to know.
type Agent interface {
	Name() string
	// Description tells the planner what this agent is for. It goes verbatim into
	// the planning prompt, so it must be written for an LLM audience.
	Description() string
	// Tools names the registered tools this agent may use. Nil means none.
	Tools() []string
	Run(ctx context.Context, in Input, emit Emit) (Output, error)
}

// Deps are the shared dependencies every agent is built with.
type Deps struct {
	LLM     *core.LLMService
	Prompts *prompts.PromptLoader
}

// AgentInfo is an agent's planner-facing summary.
type AgentInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

// ──────── LLMAgent — the shared base implementation ────────

// LLMAgent implements Agent by running the standard agentic tool loop with a
// fixed role prompt and a fixed tool subset. Every specialist in this package is
// one of these; each agent file is then just a constructor.
type LLMAgent struct {
	name      string
	desc      string
	promptTpl string   // template file name, e.g. "agent_market_data.txt"
	toolNames []string // nil = no tools
	deps      Deps
}

func (a *LLMAgent) Name() string        { return a.name }
func (a *LLMAgent) Description() string { return a.desc }

func (a *LLMAgent) Tools() []string { return a.toolNames }

// Run renders the role prompt, sends the step's task (plus any routed prior
// outputs) as a single user message, and drains the agentic loop — accumulating
// the assistant text while forwarding tool activity through emit.
func (a *LLMAgent) Run(ctx context.Context, in Input, emit Emit) (Output, error) {
	systemPrompt, err := a.deps.Prompts.GetAgentPrompt(a.promptTpl)
	if err != nil {
		return Output{}, fmt.Errorf("agent %s: render prompt: %w", a.name, err)
	}

	history := []database.Message{{Role: "user", Content: renderTask(in)}}

	// toolNames must be non-nil even when empty: nil would mean "every tool" to
	// LLMChat, whereas a toolless agent genuinely wants none. Constructors in this
	// package always supply a non-nil slice; this guards against a zero-value
	// LLMAgent silently inheriting the full tool set.
	toolNames := a.toolNames
	if toolNames == nil {
		toolNames = []string{}
	}

	streamCh, err := a.deps.LLM.LLMChat(ctx, history, core.LLMOptions{
		SystemPrompt:   systemPrompt,
		Tools:          toolNames,
		StreamThinking: true,
	})
	if err != nil {
		return Output{}, fmt.Errorf("agent %s: start completion: %w", a.name, err)
	}

	var (
		text     strings.Builder
		thinking strings.Builder
		calls    []database.Tool
	)

	for ev := range streamCh {
		switch ev.Type {
		case database.EventText:
			text.WriteString(ev.Content)
			emit("step_text", ev.Content, nil)

		case database.EventThinking:
			// Kept, not emitted: it is a fallback for an empty answer, not progress
			// the reader should see twice.
			thinking.WriteString(ev.Content)

		case database.EventToolCall:
			// Data is typed any; assert defensively so a provider quirk degrades
			// into a dropped event rather than panicking this goroutine.
			if tc, ok := ev.Data.(database.Tool); ok {
				emit("tool_call", "", tc)
			}

		case database.EventToolResult:
			if tr, ok := ev.Data.(database.Tool); ok {
				calls = append(calls, tr)
				emit("tool_result", "", tr)
			}

		case database.EventError:
			return Output{Content: text.String(), Tools: calls},
				fmt.Errorf("agent %s: %v", a.name, ev.Data)
		}
	}

	// A context deadline kills the provider stream, which closes the channel with
	// no error event — surface it rather than returning a silently empty result.
	if err := ctx.Err(); err != nil {
		return Output{Content: bestEffortContent(text.String(), thinking.String(), calls), Tools: calls},
			fmt.Errorf("agent %s: %w", a.name, err)
	}

	out := Output{Content: bestEffortContent(text.String(), thinking.String(), calls), Tools: calls}
	if out.Content == "" {
		return out, fmt.Errorf("agent %s: produced no output", a.name)
	}

	slog.Info("agent completed", "agent", a.name, "text_len", len(out.Content), "tool_calls", len(calls))
	return out, nil
}

// bestEffortContent picks the best answer the round actually produced.
//
// The straightforward reading — "the assistant's text is the answer" — loses whole
// steps with the current provider: a reasoning model regularly returns an answer
// with `content` empty and everything in the reasoning channel, and an agent that
// then reports nothing also throws away the tool results it paid for. Preference
// order is written text, then reasoning, then the raw tool output, so a downstream
// step gets something real to work from in every case but a completely silent round.
func bestEffortContent(text, thinking string, calls []database.Tool) string {
	if out := strings.TrimSpace(text); out != "" {
		return out
	}
	if out := strings.TrimSpace(thinking); out != "" {
		return out
	}
	return renderToolResults(calls)
}

// renderToolResults is the last resort: the tool output as retrieved, labelled so a
// later agent can tell it is raw data rather than an analysis.
func renderToolResults(calls []database.Tool) string {
	var sb strings.Builder
	for _, c := range calls {
		if strings.TrimSpace(c.Result) == "" {
			continue
		}
		if sb.Len() == 0 {
			sb.WriteString("Raw tool output (the agent returned no written summary):\n")
		}
		fmt.Fprintf(&sb, "\n--- %s ---\n%s\n", c.Name, c.Result)
	}
	return strings.TrimSpace(sb.String())
}

// renderTask builds the single user message an agent receives: the user's goal for
// grounding, its own task, then the outputs of whichever earlier steps were routed
// to it.
func renderTask(in Input) string {
	var sb strings.Builder

	if in.Goal != "" {
		sb.WriteString("USER'S ORIGINAL GOAL (for context and language — do not answer it directly unless that is your task):\n")
		sb.WriteString(in.Goal)
		sb.WriteString("\n\n")
	}

	sb.WriteString("TASK:\n")
	sb.WriteString(in.Task)

	if len(in.Context) == 0 {
		return sb.String()
	}

	// Sort keys so the prompt is deterministic across runs — map iteration order
	// would otherwise reshuffle the context on every call.
	ids := make([]string, 0, len(in.Context))
	for id := range in.Context {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	sb.WriteString("\n\nFINDINGS FROM EARLIER STEPS:\n")
	for _, id := range ids {
		fmt.Fprintf(&sb, "\n--- %s ---\n%s\n", id, in.Context[id])
	}
	return sb.String()
}
