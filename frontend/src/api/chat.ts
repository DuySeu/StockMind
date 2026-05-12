import api from "./index";
import type { Message } from "@/types/message";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export type ChatEventType = "start" | "thinking" | "text" | "tool_call" | "tool_result" | "done" | "error";

export interface ChatEvent {
  type: ChatEventType;
  data?: unknown;
}

/* -------------------------------------------------------------------------- */
/*  SSE parsing helpers                                                       */
/* -------------------------------------------------------------------------- */

/**
 * Parse a single SSE frame (possibly containing multiple `data:` lines)
 * and emit the parsed event via `onEvent`.
 */
function parseFrame(frame: string, onEvent: (event: ChatEvent) => void) {
  const dataLines = frame
    .split("\n")
    .filter((l) => l.startsWith("data:"))
    .map((l) => l.slice(5).replace(/^ /, ""));

  if (dataLines.length === 0) return;

  const payload = dataLines.join("\n");
  try {
    const raw = JSON.parse(payload) as ChatEvent;
    onEvent(raw);
  } catch (err) {
    console.error("Failed to parse SSE frame:", err, payload);
  }
}

/* -------------------------------------------------------------------------- */
/*  POST /chat — single endpoint: sends message & returns SSE stream          */
/* -------------------------------------------------------------------------- */
export const sendChatMessage = async (
  content: string,
  sessionId: string | undefined,
  onEvent: (event: ChatEvent) => void,
  onError: (error: unknown) => void,
  file?: File | null,
  signal?: AbortSignal,
): Promise<void> => {
  try {
    let body: BodyInit;
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
      signal,
    });

    if (!response.ok) {
      throw new Error(`Chat request failed: ${response.status}`);
    }
    if (!response.body) {
      throw new Error("Response body is null");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    // Frames are separated by a blank line. Handle both `\n\n` and `\r\n\r\n`.
    const FRAME_DELIM = /\r?\n\r?\n/;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Drain all complete frames from the buffer.
      let match = buffer.match(FRAME_DELIM);
      while (match && match.index !== undefined) {
        const frame = buffer.slice(0, match.index);
        buffer = buffer.slice(match.index + match[0].length);
        if (frame.trim()) parseFrame(frame, onEvent);
        match = buffer.match(FRAME_DELIM);
      }
    }

    // Flush any trailing frame (some servers don't send a final blank line).
    if (buffer.trim()) parseFrame(buffer, onEvent);
  } catch (error) {
    // Swallow intentional aborts so consumers don't treat them as failures.
    if ((error as { name?: string })?.name === "AbortError") return;
    onError(error);
  }
};

/* -------------------------------------------------------------------------- */
/*  GET /sessions/:id — fetch persisted messages for a session                */
/* -------------------------------------------------------------------------- */

/**
 * Normalize a raw message from the backend into the frontend Message type.
 */
function normalizeMessage(raw: any): Message {
  const role = raw.role as Message["role"];
  return { role, content: raw.content, metadata: raw.tool_calls };
}

export const getMessages = async (sessionId: string): Promise<Message[]> => {
  const response = await api.get(`/sessions/${sessionId}`);
  return (response.data as any[]).map(normalizeMessage);
};
