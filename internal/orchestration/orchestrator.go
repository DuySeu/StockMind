// Package orchestration runs a planned multi-agent pipeline: it asks a planner
// for a validated plan, then executes that plan's steps strictly in order against
// the agent registry, emitting the plan and per-step progress as events.
//
// It depends on internal/agents; agents never depends on it.
package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"stockmind/internal/agents"
)

// Default budgets, sized against the HTTP server's WriteTimeout (10 minutes).
//
// Raised from 90s/4m after a real run: a single step asked to fetch prices,
// history and three financial statements needed more than 90s of tool calls, and
// a four-step plan finished at 237s — three seconds inside the old total budget.
const (
	DefaultStepTimeout = 150 * time.Second
	DefaultTotalBudget = 8 * time.Minute
)

// Planner is the planning dependency, kept as an interface so the orchestrator is
// testable with no LLM and no network. agents.Planner satisfies it.
type Planner interface {
	Plan(ctx context.Context, goal string, roster []agents.AgentInfo) (agents.Plan, error)
}

// Options tunes a run.
type Options struct {
	// StopOnError aborts the run at the first failing step. Default false: a
	// failure is reported, its error text recorded as that step's output so later
	// steps can adapt, and execution continues. This mirrors the existing research
	// flow, which treats a failed LLM synthesis as non-fatal rather than losing
	// the work already done.
	StopOnError bool
	StepTimeout time.Duration
	TotalBudget time.Duration
}

func (o Options) withDefaults() Options {
	if o.StepTimeout <= 0 {
		o.StepTimeout = DefaultStepTimeout
	}
	if o.TotalBudget <= 0 {
		o.TotalBudget = DefaultTotalBudget
	}
	return o
}

// Orchestrator plans and executes multi-agent pipelines.
type Orchestrator struct {
	planner  Planner
	registry *agents.Registry
	opts     Options
}

// New creates an Orchestrator with default options.
func New(planner Planner, registry *agents.Registry) *Orchestrator {
	return NewWithOptions(planner, registry, Options{})
}

// NewWithOptions creates an Orchestrator with explicit options; zero fields take
// their defaults.
func NewWithOptions(planner Planner, registry *agents.Registry, opts Options) *Orchestrator {
	return &Orchestrator{planner: planner, registry: registry, opts: opts.withDefaults()}
}

// Result is the collected outcome of a run, for non-streaming callers.
type Result struct {
	Plan  agents.Plan  `json:"plan"`
	Steps []StepResult `json:"steps"`
	Final string       `json:"final"`
}

// Request is one pipeline run's input.
//
// Goal and History are separate because they are read by different consumers for
// different reasons. The planner needs the conversation to resolve a follow-up
// ("what about VNM?"), while every agent takes its cue for tone and — critically —
// *language* from Goal alone. Folding the conversation into Goal, complete with
// English section labels, is enough to make a model answer a Vietnamese question
// in another language entirely.
type Request struct {
	// Goal is the user's request, verbatim. This is what agents receive.
	Goal string
	// History is the prior conversation, already rendered. Planning-only, and
	// optional: an opening turn has none.
	History string
}

// planningGoal is what the planner is asked to decompose.
func (r Request) planningGoal() string {
	if r.History == "" {
		return r.Goal
	}
	return r.History + "\n\nCURRENT REQUEST:\n" + r.Goal
}

// Run plans and executes a pipeline for req.
//
// It returns immediately with a buffered channel and does the work in one
// goroutine, closing the channel when finished — the same shape as
// core.LLMService.LLMChat, so handlers can relay it the same way. The returned
// error covers only refusal to start; everything after that arrives as an
// EventError followed by EventDone.
func (o *Orchestrator) Run(ctx context.Context, req Request) (<-chan Event, error) {
	out := make(chan Event, 16)

	go func() {
		defer close(out)

		ctx, cancel := context.WithTimeout(ctx, o.opts.TotalBudget)
		defer cancel()

		plan, err := o.planner.Plan(ctx, req.planningGoal(), o.registry.Roster())
		if err != nil {
			out <- Event{Type: EventError, Data: fmt.Sprintf("planning failed: %v", err)}
			out <- Event{Type: EventDone}
			return
		}

		plan = ensureSynthesizer(plan, o.registry, req.Goal)
		out <- Event{Type: EventPlan, Data: plan}

		// An empty final means every step failed or the budget ran out — an
		// EventError has already gone out, so don't follow it with an empty answer.
		if final := o.execute(ctx, req.Goal, plan, out); final != "" {
			out <- Event{Type: EventFinal, Content: final}
		}
		out <- Event{Type: EventDone}
	}()

	return out, nil
}

// Collect drains a run into a Result, for the non-streaming endpoint.
func (o *Orchestrator) Collect(ctx context.Context, req Request) (Result, error) {
	ch, err := o.Run(ctx, req)
	if err != nil {
		return Result{}, err
	}

	var (
		res    Result
		runErr error
	)
	for ev := range ch {
		switch ev.Type {
		case EventPlan:
			if p, ok := ev.Data.(agents.Plan); ok {
				res.Plan = p
			}
		case EventStepDone, EventStepError:
			if sr, ok := ev.Data.(StepResult); ok {
				res.Steps = append(res.Steps, sr)
			}
		case EventFinal:
			res.Final = ev.Content
		case EventError:
			if msg, ok := ev.Data.(string); ok {
				runErr = fmt.Errorf("%s", msg)
			}
		}
	}
	if runErr != nil {
		return res, runErr
	}
	return res, nil
}

// execute runs the plan's steps in order and returns the final step's content.
func (o *Orchestrator) execute(ctx context.Context, goal string, plan agents.Plan, out chan<- Event) string {
	// outputs accumulates each completed step's content, keyed by step ID, so
	// later steps can be handed exactly the findings they asked for.
	outputs := make(map[string]string, len(plan.Steps))
	var last string

	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			out <- Event{Type: EventError, Data: fmt.Sprintf("run budget exhausted before step %q: %v", step.ID, err)}
			return last
		}

		out <- Event{Type: EventStepStart, Data: StepInfo{
			Index:  i + 1,
			Total:  len(plan.Steps),
			ID:     step.ID,
			Agent:  step.Agent,
			Task:   step.Task,
			Reason: step.Reason,
		}}

		content, err := o.runStep(ctx, goal, step, outputs, out)
		if err != nil {
			slog.Error("pipeline step failed", "step", step.ID, "agent", step.Agent, "error", err)
			out <- Event{Type: EventStepError, Data: StepResult{
				ID: step.ID, Agent: step.Agent, Content: content, Err: err.Error(),
			}}

			if o.opts.StopOnError {
				return last
			}
			// A failed step often still produced something: a step that answers in
			// full and then trips the deadline on its closing token comes back with
			// both. Discarding that threw away 26 KB of finished analysis in testing.
			// The note stays attached so the synthesizer keeps reporting the gap
			// rather than presenting partial work as complete.
			if strings.TrimSpace(content) != "" {
				outputs[step.ID] = fmt.Sprintf("[step %q did not finish cleanly: %v — partial output follows]\n%s",
					step.ID, err, content)
				last = content
				continue
			}
			// Nothing usable. A downstream step told "this lookup failed" can work
			// around it; silently substituting an empty string would let it report a
			// gap as a finding.
			outputs[step.ID] = fmt.Sprintf("[step %q failed: %v]", step.ID, err)
			continue
		}

		outputs[step.ID] = content
		last = content
		out <- Event{Type: EventStepDone, Data: StepResult{
			ID: step.ID, Agent: step.Agent, Content: content,
		}}
	}

	return last
}

// runStep executes one step under its own timeout.
func (o *Orchestrator) runStep(
	ctx context.Context,
	goal string,
	step agents.Step,
	outputs map[string]string,
	out chan<- Event,
) (string, error) {
	agent, err := o.registry.Get(step.Agent)
	if err != nil {
		// Validation already rejected unknown agents, so reaching here means the
		// plan and registry disagree — worth surfacing rather than swallowing.
		return "", err
	}

	stepCtx, cancel := context.WithTimeout(ctx, o.opts.StepTimeout)
	defer cancel()

	emit := func(kind string, content string, data any) {
		switch EventType(kind) {
		case EventStepText:
			out <- Event{Type: EventStepText, Content: content, Data: StepRef{ID: step.ID, Agent: step.Agent}}
		case EventToolCall:
			out <- Event{Type: EventToolCall, Data: data}
		case EventToolResult:
			out <- Event{Type: EventToolResult, Data: data}
		}
	}

	result, err := agent.Run(stepCtx, agents.Input{
		Task:    step.Task,
		Goal:    goal,
		Context: selectContext(outputs, step.UseOutputOf),
	}, emit)
	if err != nil {
		return result.Content, err
	}
	return result.Content, nil
}

// selectContext picks just the prior outputs a step asked for. Validation
// guarantees every reference resolves to an earlier step, so a missing key here
// would mean that step failed and its placeholder was recorded instead.
func selectContext(outputs map[string]string, refs []string) map[string]string {
	if len(refs) == 0 {
		return nil
	}
	sel := make(map[string]string, len(refs))
	for _, ref := range refs {
		if v, ok := outputs[ref]; ok {
			sel[ref] = v
		}
	}
	return sel
}

// ensureSynthesizer guarantees the plan ends with a synthesis step, so callers
// always get one coherent answer rather than whatever the last specialist said.
//
// If the registry has no synthesizer (a test roster, say), the plan is returned
// unchanged — the last step's output then becomes the final answer.
func ensureSynthesizer(plan agents.Plan, registry *agents.Registry, goal string) agents.Plan {
	if len(plan.Steps) == 0 {
		return plan
	}
	if plan.Steps[len(plan.Steps)-1].Agent == agents.SynthesizerAgentName {
		return plan
	}
	if _, err := registry.Get(agents.SynthesizerAgentName); err != nil {
		slog.Warn("no synthesizer agent registered; using last step's output as the final answer")
		return plan
	}

	refs := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		refs = append(refs, s.ID)
	}

	// The user's own words, not plan.Goal: the planner's restatement is a
	// paraphrase, usually rewritten into English, and this task line is one of the
	// few places the synthesizer sees what was actually asked.
	if goal == "" {
		goal = plan.Goal
	}
	if goal == "" {
		goal = "the user's request"
	}

	plan.Steps = append(plan.Steps, agents.Step{
		ID:          synthesisStepID(plan.Steps),
		Agent:       agents.SynthesizerAgentName,
		Task:        fmt.Sprintf("Using the findings from the earlier steps, produce the final answer to: %s", goal),
		Reason:      "Merge the findings from every step into one answer.",
		UseOutputOf: refs,
	})
	return plan
}

// synthesisStepID picks an ID that cannot collide with an existing step's.
func synthesisStepID(steps []agents.Step) string {
	const base = "synthesis"
	taken := make(map[string]struct{}, len(steps))
	for _, s := range steps {
		taken[s.ID] = struct{}{}
	}
	id := base
	for i := 2; ; i++ {
		if _, clash := taken[id]; !clash {
			return id
		}
		id = fmt.Sprintf("%s_%d", base, i)
	}
}
