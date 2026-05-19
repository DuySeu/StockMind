package server

import (
	"net/http"
	"strconv"

	"stockmind/internal/common"
	"stockmind/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

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
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Session ID is required")
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

	convID := uuid.Must(uuid.Parse(sessionID))
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
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Session ID is required")
		return
	}
	if err := s.queries.DeleteConversationByID(r.Context(), uuid.Must(uuid.Parse(sessionID))); err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}
