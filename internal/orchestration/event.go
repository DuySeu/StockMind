package orchestration

import "stockmind/internal/agents"

// EventType names a stage in a pipeline run. The values deliberately parallel
// database.StreamEventType so the frontend's existing SSE parsing generalises.
type EventType string

const (
	// EventPlan carries the validated Plan, emitted before any step runs so the
	// client can show what is about to happen. Data: Plan.
	EventPlan EventType = "plan"
	// EventStepStart marks a step beginning. Data: StepInfo.
	EventStepStart EventType = "step_start"
	// EventToolCall / EventToolResult mirror the chat path's tool events.
	// Data: database.Tool.
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	// EventStepText is a text delta from the running step. Content: the delta.
	// Data: StepRef, so a client can attribute the delta to its step.
	EventStepText EventType = "step_text"
	// EventStepDone carries a completed step's output. Data: StepResult.
	EventStepDone EventType = "step_done"
	// EventStepError reports a failed step. Data: StepResult with Err set.
	EventStepError EventType = "step_error"
	// EventFinal carries the answer. Content: the text. Data: nil.
	EventFinal EventType = "final"
	// EventError reports a run-level failure (planning failed, budget exhausted).
	// Data: string.
	EventError EventType = "error"
	// EventDone terminates the stream.
	EventDone EventType = "done"
)

// Event is one emission from a pipeline run.
type Event struct {
	Type    EventType `json:"type"`
	Content string    `json:"content,omitempty"`
	Data    any       `json:"data,omitempty"`
}

// Plan is re-exported so SSE consumers and handlers need not import agents.
type Plan = agents.Plan

// StepInfo describes a step that is starting.
type StepInfo struct {
	Index  int    `json:"index"` // 1-based position in the plan
	Total  int    `json:"total"`
	ID     string `json:"id"`
	Agent  string `json:"agent"`
	Task   string `json:"task"`
	Reason string `json:"reason"`
}

// StepRef identifies the step a text delta belongs to.
type StepRef struct {
	ID    string `json:"id"`
	Agent string `json:"agent"`
}

// StepResult is a finished step, successful or not.
type StepResult struct {
	ID      string `json:"id"`
	Agent   string `json:"agent"`
	Content string `json:"content"`
	Err     string `json:"error,omitempty"`
}
