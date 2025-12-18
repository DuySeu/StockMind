import api from ".";

export const getSessions = async (): Promise<any[]> => {
  const response = await api.get("/sessions");
  return response.data;
};

export const deleteSession = async (id: string): Promise<void> => {
  await api.delete(`/sessions/${id}`);
};
