package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

func (s *Server) GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	// Get users from database
	sessions, err := s.db.GetSessionsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get sessions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Return sessions list
	if err := json.NewEncoder(w).Encode(sessions); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetMessagesBySessionIdHandler(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL parameter
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Get messages from database
	messages, err := s.db.GetSessionHistoryBySessionID(r.Context(), uuid.Must(uuid.Parse(sessionID)))
	if err != nil {
		http.Error(w, "Failed to get messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	msgs := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Content.OfOpenAI != nil {
			msgs = append(msgs, *msg.Content.OfOpenAI)
		}
	}

	// Return messages list
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL parameter
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Delete session from database
	if err := s.db.DeleteSessionByID(r.Context(), uuid.Must(uuid.Parse(sessionID))); err != nil {
		http.Error(w, "Failed to delete session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Return success response
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Session deleted successfully"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
