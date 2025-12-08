package server

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
