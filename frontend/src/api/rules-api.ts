import axiosClient from ".";

export const getUserInfo = async () => {
  const response = await axiosClient.get(`me`);
  return response;
};

export const getRules = async (): Promise<any> => {
  const response = await axiosClient.get(`rules`);
  return response;
};

export const getRuleById = async (id: string): Promise<any> => {
  const response = await axiosClient.get(`rules/${id}`);
  return response;
};

export const createRule = async (data: any): Promise<any> => {
  const response = await axiosClient.post(`rules`, data);
  return response;
};

export const updateRule = async (id: string, data: any): Promise<any> => {
  const response = await axiosClient.put(`rules/${id}`, data);
  return response;
};

export const deleteRule = async (id: string): Promise<any> => {
  const response = await axiosClient.delete(`rules/${id}`);
  return response;
};
