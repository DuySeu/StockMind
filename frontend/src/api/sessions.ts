import api from ".";

export const getSession = async () => {
  const response = await api.get("/sessions");
  return response.data;
};
