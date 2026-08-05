package server

import (
	"net/http"
	"stockmind/internal/common"
)

// GET /v1/agent_flows - List the configured agent flows
func (s *Server) ListAgentFlowsHandler(w http.ResponseWriter, r *http.Request) {
	flows, err := s.queries.ListAgentFlows(r.Context())
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get agent flows: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, flows)
}

// POST /v1/agent_flows - Create an agent flow (not implemented)
func (s *Server) CreateAgentFlowHandler(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "Create agent flow"})
}
