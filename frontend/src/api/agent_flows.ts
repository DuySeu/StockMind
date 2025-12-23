import type { AgentFlow } from "@/types/agent_flow";
import api from ".";

export const getAgentFlows = async (): Promise<AgentFlow[]> => {
  const response = await api.get("/agent_flows");
  return response.data;
};
