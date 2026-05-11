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

export const startChatSession = async (
  content: string,
  sessionId: string | undefined,
  file?: File | null
): Promise<string> => {
  let body;
  const headers: Record<string, string> = {};

  if (file) {
    const formData = new FormData();
    formData.append("content", content);
    if (sessionId) formData.append("session_id", sessionId);
    formData.append("file", file);
    body = formData;
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

  const data = await response.json();
  return data.session_id;
};

export const streamChatSession = async (
  sessionId: string,
  onMessage: (data: ChatResponse) => void,
  onError: (error: any) => void
) => {
  try {
    const response = await fetch(`${api.defaults.baseURL}/chat/stream?session_id=${sessionId}`, {
      method: "GET",
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

/**
 * Normalize a raw message from the backend (openai.ChatCompletionMessage shape)
 * into the frontend Message type where content is always ContentPart[].
 */
function normalizeMessage(raw: any): Message {
  const role = raw.role as Message["role"];

  // Already in the expected shape (ContentPart[])
  if (Array.isArray(raw.content)) {
    return { role, content: raw.content, tool_calls: raw.tool_calls };
  }

  // Backend returns content as a plain string — wrap it
  const parts: any[] = [];
  if (typeof raw.content === "string" && raw.content) {
    parts.push({ type: "text", text: raw.content });
  }

  return { role, content: parts, tool_calls: raw.tool_calls };
}

export const getMessages = async (sessionId: string): Promise<Message[]> => {
  const response = await api.get(`/sessions/${sessionId}`);
  return (response.data as any[]).map(normalizeMessage);
};
