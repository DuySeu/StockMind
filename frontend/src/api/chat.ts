import api from "./index";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export interface ChatResponse {
  type: "thinking_delta" | "text_delta" | "tool_use" | "tool_result" | "complete";
  data?: {
    thinking?: string;
    text?: string;
    session_id?: string;
    tool_calls?: Record<string, any>;
    result?: Record<string, any>;
  };
}

export const chatWithLLM = async (
  content: string,
  sessionId: string | undefined,
  onMessage: (data: ChatResponse) => void,
  onError: (error: any) => void,
  file?: File | null
) => {
  try {
    let body;
    const headers: Record<string, string> = {};

    if (file) {
      const formData = new FormData();
      formData.append("content", content);
      if (sessionId) formData.append("session_id", sessionId);
      formData.append("file", file);
      body = formData;
      // Do NOT set Content-Type header for FormData, browser does it with boundary
    } else {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify({ content, session_id: sessionId });
    }

    const response = await fetch(`${api.defaults.baseURL}/chat`, {
      method: "POST",
      headers,
      body,
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    if (!response.body) {
      throw new Error("Response body is null");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          try {
            // Handle potential multiple JSON objects in one line if backend flushes weirdly,
            // though \n\n split should handle standard SSE.
            // The backend sends: fmt.Fprintf(w, "data: %s\n\n", data)
            const parsed = JSON.parse(data) as ChatResponse;
            onMessage(parsed);
          } catch (e) {
            console.error("Error parsing SSE data:", e);
          }
        }
      }
    }
  } catch (error) {
    onError(error);
  }
};

import type { Message } from "@/types/message";

export const getMessages = async (sessionId: string): Promise<Message[]> => {
  const response = await api.get(`/sessions/${sessionId}`);
  return response.data;
};
