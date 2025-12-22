import api from ".";

export const getAgentFlows = async (): Promise<any[]> => {
  const response = await api.get("/agent_flows");
  return response.data;
};
