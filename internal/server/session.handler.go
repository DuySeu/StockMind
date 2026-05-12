package server

import (
	"net/http"
	"stockmind/internal/common"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /v1/sessions - Get sessions
func (s *Server) GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	// Get users from database
	sessions, err := s.db.GetSessionsByUserID(r.Context(), userID)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
		return
	}

	// Return sessions list
	common.WriteJSON(w, http.StatusOK, sessions)
}

func (s *Server) GetMessagesBySessionIdHandler(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL parameter
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Get messages from database
	messages, err := s.db.GetSessionHistoryBySessionID(r.Context(), uuid.Must(uuid.Parse(sessionID)))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get messages: "+err.Error())
		return
	}

	// Return messages list
	common.WriteJSON(w, http.StatusOK, messages)
}

func (s *Server) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL parameter
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Delete session from database
	if err := s.db.DeleteSessionByID(r.Context(), uuid.Must(uuid.Parse(sessionID))); err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}

	// Return success response
	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}
