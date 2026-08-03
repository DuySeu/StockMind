package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// maxTitleRunes caps a conversation title. Counted in runes, not bytes, so a
// Vietnamese title isn't truncated mid-character.
const maxTitleRunes = 120

// ──────── Session ID guards ────────

// parseSessionID reads the {id} URL param and parses it as a UUID, answering
// with 400 on a malformed value. The previous `uuid.Must(uuid.Parse(...))`
// panicked on any non-UUID path segment, which closed the connection without a
// response instead of returning an error the client could read.
func parseSessionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "id")
	if raw == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Session ID is required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid session ID: expected a UUID")
		return uuid.Nil, false
	}
	return id, true
}

// conversationExists reports whether a conversation row is present. A missing
// row is not an error — it's the `false` case.
func (s *Server) conversationExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if _, err := s.queries.GetConversationByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// requireConversation guarantees the session exists before a handler reads or
// writes anything under it, so callers never surface a raw driver error like
// "no rows in result set". Writes 404 / 500 and returns false when it doesn't.
func (s *Server) requireConversation(w http.ResponseWriter, r *http.Request, id uuid.UUID) bool {
	ok, err := s.conversationExists(r.Context(), id)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to look up session: "+err.Error())
		return false
	}
	if !ok {
		common.WriteJSONError(w, http.StatusNotFound, "Session not found")
		return false
	}
	return true
}

// GET /v1/sessions - Get sessions
func (s *Server) GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	sessions, err := s.queries.GetConversationsByUserID(r.Context(), userID)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, sessions)
}

func (s *Server) GetMessagesBySessionIdHandler(w http.ResponseWriter, r *http.Request) {
	convID, ok := parseSessionID(w, r)
	if !ok {
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	// 404 for an unknown session rather than an empty 200, so the client can
	// tell "this conversation is gone" from "this conversation has no messages".
	if !s.requireConversation(w, r, convID) {
		return
	}

	ctx := r.Context()

	total, err := s.queries.GetMessageCountByConversationID(ctx, convID)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to count messages: "+err.Error())
		return
	}

	messages, err := s.queries.GetMessagesByConversationID(ctx, database.GetMessagesByConversationIDParams{
		ConversationID: convID,
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get messages: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
		"has_more": int64(offset+limit) < total,
	})
}

func (s *Server) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseSessionID(w, r)
	if !ok {
		return
	}
	if !s.requireConversation(w, r, sessionID) {
		return
	}
	if err := s.queries.DeleteConversationByID(r.Context(), sessionID); err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}

type updateSessionRequest struct {
	Title string `json:"title"`
}

// PATCH /v1/sessions/{id} - Rename a conversation.
func (s *Server) UpdateSessionTitleHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseSessionID(w, r)
	if !ok {
		return
	}

	var body updateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	if runes := []rune(title); len(runes) > maxTitleRunes {
		title = string(runes[:maxTitleRunes])
	}

	if !s.requireConversation(w, r, sessionID) {
		return
	}
	if err := s.queries.UpdateConversationName(r.Context(), database.UpdateConversationNameParams{
		ID:    sessionID,
		Title: title,
	}); err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to rename session: "+err.Error())
		return
	}

	// Echo the stored title back: the server may have truncated it, and the
	// client should render what was actually saved.
	common.WriteJSON(w, http.StatusOK, map[string]string{"id": sessionID.String(), "title": title})
}
