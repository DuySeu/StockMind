package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/orchestration"
)

var (
	errInvalidJSON  = errors.New("Invalid JSON")
	errGoalRequired = errors.New("goal is required")
)

// pipelineRequest is the inbound payload for both pipeline endpoints.
type pipelineRequest struct {
	Goal string `json:"goal"`
	// SessionID is accepted and echoed for client correlation. The pipeline is
	// stateless in this version: nothing is persisted.
	SessionID string `json:"session_id,omitempty"`
}

func decodePipelineRequest(r *http.Request) (pipelineRequest, error) {
	var req pipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, errInvalidJSON
	}
	req.Goal = strings.TrimSpace(req.Goal)
	if req.Goal == "" {
		return req, errGoalRequired
	}
	return req, nil
}

// AgentPipelineHandler runs a planned multi-agent pipeline and returns the whole
// result at once.
//
// POST /v1/agent/pipeline  body: {"goal": "..."}
func (s *Server) AgentPipelineHandler(w http.ResponseWriter, r *http.Request) {
	if s.orchestrator == nil {
		common.WriteJSONError(w, http.StatusServiceUnavailable, "agent pipeline is not configured")
		return
	}

	req, err := decodePipelineRequest(r)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.orchestrator.Collect(r.Context(), req.Goal)
	if err != nil {
		// A run can fail after producing partial output; return what there is so
		// the caller can see how far it got rather than just an opaque 500.
		common.WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"error":      err.Error(),
			"session_id": req.SessionID,
			"plan":       result.Plan,
			"steps":      result.Steps,
			"final":      result.Final,
		})
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": req.SessionID,
		"plan":       result.Plan,
		"steps":      result.Steps,
		"final":      result.Final,
	})
}

// AgentPipelineStreamHandler runs a planned multi-agent pipeline, streaming the
// plan and per-step progress as SSE.
//
// POST /v1/agent/pipeline/stream  body: {"goal": "..."}
func (s *Server) AgentPipelineStreamHandler(w http.ResponseWriter, r *http.Request) {
	if s.orchestrator == nil {
		common.WriteJSONError(w, http.StatusServiceUnavailable, "agent pipeline is not configured")
		return
	}

	req, err := decodePipelineRequest(r)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// "start" mirrors the chat handler's opening frame so clients can key off the
	// same envelope shape.
	common.WriteSSE(w, common.SSEEvent("start", map[string]any{"session_id": req.SessionID}))

	stream, err := s.orchestrator.Run(r.Context(), req.Goal)
	if err != nil {
		common.WriteSSE(w, common.SSEEvent(database.EventError, err.Error()))
		return
	}

	// If we return early (client gone), keep draining so the orchestrator's
	// goroutine can finish instead of blocking forever on a full channel buffer.
	// On the normal path the range below has already drained it and this exits at once.
	defer func() {
		go func() {
			//nolint:revive // intentionally empty: discard remaining events
			for range stream {
			}
		}()
	}()

	for ev := range stream {
		// Text-bearing events put their payload in Content, like the chat path's
		// text events; everything else carries a structured Data payload.
		switch ev.Type {
		case orchestration.EventStepText:
			common.WriteSSE(w, map[string]any{
				"type":    ev.Type,
				"content": ev.Content,
				"data":    ev.Data,
			})
		case orchestration.EventFinal:
			common.WriteSSE(w, map[string]any{
				"type":    ev.Type,
				"content": ev.Content,
			})
		default:
			common.WriteSSE(w, common.SSEEvent(database.StreamEventType(ev.Type), ev.Data))
		}

		// Stop writing into a dead connection. The orchestrator's own goroutine
		// unwinds via r.Context() cancellation.
		if r.Context().Err() != nil {
			slog.Info("agent pipeline: client disconnected", "session_id", req.SessionID)
			return
		}
	}
}
