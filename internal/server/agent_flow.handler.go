package server

import (
	"net/http"
	"stockmind/internal/common"
)

func (s *Server) ListAgentFlowsHandler(w http.ResponseWriter, r *http.Request) {
	flows, err := s.queries.ListAgentFlows(r.Context())
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get agent flows: "+err.Error())
		return
	}

	// Return sessions list
	common.WriteJSON(w, http.StatusOK, flows)
}

func (s *Server) CreateAgentFlowHandler(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Create agent flow"})
}
