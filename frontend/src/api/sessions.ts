import api from ".";
import type { Message } from "@/types/message";

export const getSessions = async (): Promise<any[]> => {
  const response = await api.get("/sessions");
  return response.data;
};

export const deleteSession = async (id: string): Promise<void> => {
  await api.delete(`/sessions/${id}`);
};

function normalizeMessage(raw: any): Message {
  return { role: raw.role as Message["role"], content: raw.content, metadata: raw.metadata };
}

export interface MessagesPage {
  messages: Message[];
  has_more: boolean;
}

export const getSessionMessages = async (
  id: string,
  limit = 20,
  offset = 0,
): Promise<MessagesPage> => {
  const response = await api.get(`/sessions/${id}`, { params: { limit, offset } });
  const data = response.data as { messages: any[]; has_more: boolean };
  return {
    messages: (data.messages ?? []).map(normalizeMessage),
    has_more: data.has_more ?? false,
  };
};
