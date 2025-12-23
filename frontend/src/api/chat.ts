import api from "./index";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export interface ChatResponse {
  type: "thinking_delta" | "text_delta" | "complete";
  data?: {
    thinking?: string;
    text?: string;
    session_id?: string;
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
  return response.data.map((msg: any) => {
    let content: any[] = [];

    // Check if content is already an array (from database)
    if (Array.isArray(msg.content)) {
      content = msg.content.map((part: any) => {
        if (part.type === "text") {
          return { type: "text", text: part.text || "" };
        } else if (part.type === "image_url" && part.image_url) {
          return { type: "image_url", image_url: { url: part.image_url.url } };
        }
        return part;
      });
    } else if (msg.MultiContent && msg.MultiContent.length > 0) {
      // Fallback for MultiContent (Go struct field)
      content = msg.MultiContent.map((part: any) => {
        if (part.type === "text") {
          return { type: "text", text: part.text || "" };
        } else if (part.type === "image_url" && part.image_url) {
          return { type: "image_url", image_url: { url: part.image_url.url } };
        }
        return part;
      });
    } else if (msg.content && typeof msg.content === "string") {
      // Simple string content
      content = [{ type: "text", text: msg.content }];
    }

    return {
      role: msg.role,
      content: content,
      tool_calls: msg.tool_calls,
    };
  });
};
