import axiosClient from ".";

export const getDocuments = async (): Promise<any> => {
  const response = await axiosClient.get("documents");
  return response;
};

export const getDocumentById = async (id: string): Promise<any> => {
  const response = await axiosClient.get(`documents/${id}`);
  return response;
};

export const createDocument = async (data: any): Promise<any> => {
  const response = await axiosClient.post("documents", data, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
  return response;
};

export const updateDocument = async (id: string, data: any): Promise<any> => {
  const response = await axiosClient.put(`documents/${id}`, data);
  return response;
};

export const deleteDocument = async (id: string): Promise<any> => {
  const response = await axiosClient.delete(`documents/${id}`);
  return response;
};

export const executeFlow = async (flow_id: string): Promise<any> => {
  const response = await axiosClient.post(`flow/${flow_id}/execute`);
  return response;
};
