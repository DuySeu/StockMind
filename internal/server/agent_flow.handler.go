package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) ListAgentFlowsHandler(w http.ResponseWriter, r *http.Request) {
	flows, err := s.db.ListAgentFlows(r.Context())
	if err != nil {
		http.Error(w, "Failed to get agent flows: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Return sessions list
	if err := json.NewEncoder(w).Encode(flows); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) CreateAgentFlowHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Create agent flow"}`))
}
