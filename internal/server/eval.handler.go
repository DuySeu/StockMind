package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"stockmind/internal/agents"
	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/guardrail"
	"stockmind/internal/orchestration"
	"stockmind/internal/trace"
)

// Each eval endpoint runs one module of the chat path on its own, with the same
// Go code the chat path calls, and streams its spans like a traced chat turn.
// Nothing here touches the database or the guardrail unless the module is the
// guardrail itself.

type evalMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type evalGuardrailRequest struct {
	Message string        `json:"message"`
	History []evalMessage `json:"history"`
}

type evalSanitizeRequest struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

type evalGoalRequest struct {
	Goal string `json:"goal"`
}

type evalAgentRequest struct {
	Agent   string            `json:"agent"`
	Task    string            `json:"task"`
	Goal    string            `json:"goal"`
	Context map[string]string `json:"context"`
}

type evalToolRequest struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

type evalToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

// GET /v1/eval/catalog - List the agents and tools the eval forms can pick from
func (s *Server) EvalCatalogHandler(w http.ResponseWriter, r *http.Request) {
	all := s.tools.All()
	toolList := make([]evalToolInfo, 0, len(all))
	for _, t := range all {
		toolList = append(toolList, evalToolInfo{Name: t.Name, Description: t.Description, Schema: t.Schema})
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{
		"agents":           s.registry.Roster(),
		"tools":            toolList,
		"guardrailEnabled": s.guard != nil,
	})
}

// POST /v1/eval/guardrail - Redact and classify one message, nothing after
func (s *Server) EvalGuardrailHandler(w http.ResponseWriter, r *http.Request) {
	var in evalGuardrailRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Message, "message") {
		return
	}
	s.runEval(w, r, "guardrail", in, func(ctx context.Context) (any, error) {
		masked, kinds := redactTraced(ctx, in.Message)
		out := map[string]any{"redacted": masked, "kinds": kinds}
		if s.guard == nil {
			out["note"] = "classifier off: GUARDRAIL_ENABLED=false"
			return out, nil
		}

		// Classify the masked message against the given history, as the chat path does
		history := make([]database.Message, 0, len(in.History)+1)
		for _, m := range in.History {
			history = append(history, database.Message{Role: m.Role, Content: m.Content})
		}
		history = append(history, database.Message{Role: "user", Content: masked})
		decision, err := s.guard.Classify(ctx, history)
		out["decision"] = decision
		if err != nil {
			// Fail-open is the result under test, not a failed eval
			out["fail_open"] = err.Error()
		}
		return out, nil
	})
}

// POST /v1/eval/sanitize - Clean one tool result and show what the model receives
func (s *Server) EvalSanitizeHandler(w http.ResponseWriter, r *http.Request) {
	var in evalSanitizeRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Source, "source") || !requireEval(w, in.Content, "content") {
		return
	}
	s.runEval(w, r, "sanitize", in, func(ctx context.Context) (any, error) {
		cleaned := guardrail.Sanitize(in.Source, in.Content)
		return map[string]any{
			"result":  cleaned,
			"changed": cleaned != in.Content,
			"wrapped": guardrail.WrapToolOutput(in.Source, cleaned),
		}, nil
	})
}

// POST /v1/eval/plan - Ask the planner for a plan without running any agent
func (s *Server) EvalPlanHandler(w http.ResponseWriter, r *http.Request) {
	var in evalGoalRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Goal, "goal") {
		return
	}
	s.runEval(w, r, "plan", in, func(ctx context.Context) (any, error) {
		return s.planner.Plan(ctx, in.Goal, s.registry.Roster())
	})
}

// POST /v1/eval/pipeline - Plan and run every agent, skipping the guardrail
func (s *Server) EvalPipelineHandler(w http.ResponseWriter, r *http.Request) {
	var in evalGoalRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Goal, "goal") {
		return
	}
	s.runEval(w, r, "pipeline", in, func(ctx context.Context) (any, error) {
		return s.orchestrator.Collect(ctx, orchestration.Request{Goal: in.Goal})
	})
}

// POST /v1/eval/agent - Run one agent on one task
func (s *Server) EvalAgentHandler(w http.ResponseWriter, r *http.Request) {
	var in evalAgentRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Agent, "agent") || !requireEval(w, in.Task, "task") {
		return
	}
	agent, err := s.registry.Get(in.Agent)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.runEval(w, r, "agent", in, func(ctx context.Context) (any, error) {
		goal := in.Goal
		if goal == "" {
			goal = in.Task
		}
		return agent.Run(ctx, agents.Input{Task: in.Task, Goal: goal, Context: in.Context}, func(string, string, any) {})
	})
}

// POST /v1/eval/tool - Execute one tool, native or MCP-bridged, with raw args
func (s *Server) EvalToolHandler(w http.ResponseWriter, r *http.Request) {
	var in evalToolRequest
	if !decodeEval(w, r, &in) || !requireEval(w, in.Tool, "tool") {
		return
	}
	args := strings.TrimSpace(string(in.Args))
	if args == "" || args == "null" {
		args = "{}"
	}
	s.runEval(w, r, "tool", in, func(ctx context.Context) (any, error) {
		result, err := s.tools.Execute(ctx, in.Tool, args)
		if err != nil {
			return nil, err
		}
		if json.Valid([]byte(result)) {
			return json.RawMessage(result), nil
		}
		return result, nil
	})
}

// runEval streams one module run: spans as `trace` events while it works, then
// `result` or `error`, then `done`. Errors go out verbatim, since the eval page
// exists to show them.
func (s *Server) runEval(w http.ResponseWriter, r *http.Request, module string, input any, run func(ctx context.Context) (any, error)) {
	rec := trace.NewRecorder(traceBuffer)
	ctx, end := trace.Start(trace.WithRecorder(r.Context(), rec), "eval", "eval."+module, input)
	tt := &turnTrace{rec: rec, end: end}
	startSSE(w)

	// Run the module; the buffered channel lets it finish after the client leaves
	type outcome struct {
		output any
		err    error
	}
	doneCh := make(chan outcome, 1)
	go func() {
		output, err := run(ctx)
		doneCh <- outcome{output, err}
	}()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-heartbeat.C:
			if err := common.WriteSSEComment(w, "ping"); err != nil {
				return
			}

		case ev := <-tt.events():
			heartbeat.Reset(sseHeartbeatInterval)
			if err := common.WriteSSE(w, common.SSEEvent(database.EventTrace, ev)); err != nil {
				return
			}

		case o := <-doneCh:
			tt.finish(w, o.output, o.err)
			if o.err != nil {
				_ = common.WriteSSE(w, common.SSEEvent(database.EventError, map[string]any{"message": o.err.Error()}))
			} else {
				_ = common.WriteSSE(w, common.SSEEvent(database.EventResult, o.output))
			}
			_ = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{}))
			return
		}
	}
}

// decodeEval reads a JSON body into v, answering 400 itself when it cannot.
func decodeEval(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// requireEval answers 400 when a required field is blank.
func requireEval(w http.ResponseWriter, value, field string) bool {
	if strings.TrimSpace(value) == "" {
		common.WriteJSONError(w, http.StatusBadRequest, field+" is required")
		return false
	}
	return true
}
