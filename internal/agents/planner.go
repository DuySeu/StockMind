package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"stockmind/internal/llm/prompts"
)

// planRepairAttempts is how many times the planner re-prompts after a validation
// failure. One is deliberate: the active provider offers no JSON-Schema
// enforcement, so a single corrective round-trip is worth it, but a model that
// fails twice with the errors spelled out is not going to succeed on a third try.
const planRepairAttempts = 1

// Planner turns a natural-language goal into a validated Plan.
//
// It is not an LLMAgent: it produces structured JSON in one non-streaming call
// rather than streaming prose through the tool loop.
type Planner struct {
	deps Deps
}

// NewPlanner creates the planning agent.
func NewPlanner(d Deps) *Planner {
	return &Planner{deps: d}
}

func (p *Planner) Name() string { return "planner" }

func (p *Planner) Description() string {
	return "Decomposes a goal into an ordered list of steps delegated to specialist agents."
}

// Plan generates a plan for goal and validates it against roster, re-prompting
// once with the validation errors if the first attempt is rejected.
func (p *Planner) Plan(ctx context.Context, goal string, roster []AgentInfo) (Plan, error) {
	if strings.TrimSpace(goal) == "" {
		return Plan{}, fmt.Errorf("planner: goal is empty")
	}

	names := make([]string, 0, len(roster))
	for _, a := range roster {
		names = append(names, a.Name)
	}

	rendered := RenderRoster(roster)
	var lastErr error

	for attempt := 0; attempt <= planRepairAttempts; attempt++ {
		var errText string
		if lastErr != nil {
			errText = lastErr.Error()
		}

		prompt, err := p.deps.Prompts.GetPlanPrompt(prompts.PlanParams{
			Goal:   goal,
			Roster: rendered,
			Errors: errText,
		})
		if err != nil {
			return Plan{}, fmt.Errorf("planner: render prompt: %w", err)
		}

		var plan Plan
		if err := p.deps.LLM.StructuredCompletion(ctx, prompt, &plan); err != nil {
			// A transport or parse failure is not something the repair prompt can
			// fix — the model never produced a plan to correct. Fail fast.
			return Plan{}, fmt.Errorf("planner: completion (attempt %d): %w", attempt+1, err)
		}

		if err := plan.Validate(names); err != nil {
			lastErr = err
			slog.Warn("planner: plan rejected", "attempt", attempt+1, "error", err)
			continue
		}

		if plan.Goal == "" {
			plan.Goal = goal
		}
		slog.Info("planner: plan accepted", "attempt", attempt+1,
			"steps", len(plan.Steps), "agents", strings.Join(plan.AgentNames(), ","))
		return plan, nil
	}

	return Plan{}, fmt.Errorf("planner: no valid plan after %d attempts: %w", planRepairAttempts+1, lastErr)
}

// planJSONExample is the contract shown in plan_prompt.txt. It is unmarshalled in
// a test to guarantee the documented example stays parseable into a Plan — the
// prompt is the only schema the model gets, so it must not drift from the struct.
var planJSONExample = json.RawMessage(`{
  "goal": "one sentence",
  "steps": [
    {"id": "s1", "agent": "market_data", "task": "…", "reason": "…", "use_output_of": []}
  ]
}`)
