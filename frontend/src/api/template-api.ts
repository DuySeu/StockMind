import axiosClient from ".";

export const getUserInfo = async () => {
  const response = await axiosClient.get(`me`);
  return response;
};

export const getTemplates = async (): Promise<any> => {
  const response = await axiosClient.get(`templates`);
  return response;
};

export const getTemplateById = async (id: string): Promise<any> => {
  const response = await axiosClient.get(`templates/${id}`);
  return response;
};

export const createTemplate = async (data: any): Promise<any> => {
  const response = await axiosClient.post(`templates`, data);
  return response;
};

export const updateTemplate = async (id: string, data: any): Promise<any> => {
  const response = await axiosClient.put(`templates/${id}`, data);
  return response;
};

export const deleteTemplate = async (id: string): Promise<any> => {
  const response = await axiosClient.delete(`templates/${id}`);
  return response;
};
