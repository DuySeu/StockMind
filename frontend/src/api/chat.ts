import { QUOTA_ERROR_CODE } from "@/types/message";
import api from "./index";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export type ChatEventType = "start" | "thinking" | "text" | "tool_call" | "tool_result" | "done" | "error";

export interface ChatEvent {
  type: ChatEventType;
  data?: unknown;
}

/**
 * A failed `/chat` request, carrying the message the server actually sent.
 * Handlers write `{"error": "..."}` for every 4xx/5xx, and throwing away that
 * body left the UI showing a generic "an error occurred" for causes as
 * different as an unknown session and an unsupported Content-Type.
 */
export class ChatRequestError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ChatRequestError";
    this.status = status;
  }
}

/**
 * Failure code for a rejected request, so the UI can treat a quota refusal as its
 * own case. Quota normally arrives as an in-stream `error` event carrying the
 * server's code; this covers the other shape — a 402/429 from the app or from a
 * gateway in front of it, which never reaches the stream at all.
 */
export const requestFailureCode = (error: unknown): string | undefined => {
  if (!(error instanceof ChatRequestError)) return undefined;
  return error.status === 402 || error.status === 429 ? QUOTA_ERROR_CODE : undefined;
};

/** Pull `{"error": "..."}` out of a non-2xx response, falling back to the status. */
async function readErrorBody(response: Response): Promise<string> {
  try {
    const text = await response.text();
    if (!text) return `Chat request failed (${response.status})`;
    try {
      const parsed = JSON.parse(text) as { error?: string; message?: string };
      const message = parsed.error ?? parsed.message;
      if (message) return message;
    } catch {
      // Not JSON (e.g. a proxy error page) — fall through to the raw text.
    }
    return text.slice(0, 300);
  } catch {
    return `Chat request failed (${response.status})`;
  }
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
  maxMode: boolean,
  file: File | null,
  onEvent: (event: ChatEvent) => void,
  onError: (error: unknown) => void,
  signal?: AbortSignal,
): Promise<void> => {
  try {
    let body: BodyInit;
    const headers: Record<string, string> = {};

    if (file) {
      const formData = new FormData();
      formData.append("content", content);
      if (sessionId) formData.append("session_id", sessionId);
      if (maxMode) formData.append("max_mode", maxMode.toString());
      formData.append("file", file);
      body = formData;
    } else {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify({ content, session_id: sessionId, max_mode: maxMode });
    }

    const response = await fetch(`${api.defaults.baseURL}/chat`, {
      method: "POST",
      headers,
      body,
      signal,
    });

    if (!response.ok) {
      throw new ChatRequestError(response.status, await readErrorBody(response));
    }
    if (!response.body) {
      throw new ChatRequestError(response.status, "Server returned an empty response body");
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

// `getMessages`/`normalizeMessage` used to live here, reading `raw.tool_calls` —
// a field the backend has never sent. Session messages are loaded by
// `getSessionMessages` in ./sessions, which is the paginated shape the API
// actually returns.
