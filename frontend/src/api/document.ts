import api from ".";
import type { Document } from "../types/document";

export const getDocuments = async (): Promise<Document[]> => {
  const response = await api.get('/documents');
  return response.data;
};

export const getDocumentById = async (id: string): Promise<Document> => {
  const response = await api.get(`/documents/${id}`);
  return response.data;
};

export const uploadDocument = async (file: File, strategy: string): Promise<Document> => {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("strategy", strategy);

  const response = await api.post('/documents', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
  return response.data;
};

export const deleteDocument = async (id: string): Promise<void> => {
  await api.delete(`/documents/${id}`);
};
