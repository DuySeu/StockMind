import axios from "axios";
import api from ".";
import type { Message, Metadata, ToolCallContent } from "@/types/message";
import { statusFromResult, toIsError } from "@/types/message";

export const getSessions = async (): Promise<any[]> => {
  const response = await api.get("/sessions");
  return response.data;
};

export const deleteSession = async (id: string): Promise<void> => {
  await api.delete(`/sessions/${id}`);
};

/** PATCH /sessions/:id — persist a conversation title. Returns the stored title. */
export const updateSessionTitle = async (id: string, title: string): Promise<string> => {
  const response = await api.patch(`/sessions/${id}`, { title });
  return (response.data as { title?: string }).title ?? title;
};

/** True when an axios rejection is the server saying the session is gone. */
export const isNotFoundError = (error: unknown): boolean =>
  axios.isAxiosError(error) && error.response?.status === 404;

/** Server message from an axios rejection, for surfacing instead of "Request failed". */
export const errorMessage = (error: unknown, fallback: string): string => {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { error?: string } | undefined;
    return data?.error ?? error.message ?? fallback;
  }
  return error instanceof Error ? error.message : fallback;
};

/* -------------------------------------------------------------------------- */
/*  Persisted → in-memory normalization                                       */
/* -------------------------------------------------------------------------- */

/**
 * A message's persisted `metadata` is `[]database.Metadata` — an array of
 * *containers*, each with a `tools` / `attachments` / `sources` bucket:
 *
 *   [{ "tools": [{ id, name, arguments, result, is_error }] }]
 *
 * The streaming path, by contrast, appends one flat entry per tool call. Left
 * unflattened, the container objects match no `Metadata` variant, so every tool
 * chip silently vanished the moment a conversation was reloaded.
 *
 * This flattens the persisted form into the same flat entries the stream
 * produces, in stored order.
 */
function flattenMetadata(raw: unknown): Metadata[] {
  if (!Array.isArray(raw)) return [];

  const out: Metadata[] = [];
  for (const container of raw) {
    if (!container || typeof container !== "object") continue;

    const tools = (container as { tools?: unknown }).tools;
    if (Array.isArray(tools)) {
      for (const tool of tools) {
        if (!tool || typeof tool !== "object") continue;
        const t = tool as Record<string, unknown>;
        if (typeof t.id !== "string" || !t.id) continue;

        const isError = toIsError(t.is_error);
        const hasResult = typeof t.result === "string" && t.result !== "";
        out.push({
          type: "tool_call",
          id: t.id,
          name: typeof t.name === "string" ? t.name : "unknown",
          arguments: typeof t.arguments === "string" ? t.arguments : undefined,
          result: hasResult ? (t.result as string) : undefined,
          is_error: isError,
          // A persisted tool has finished by definition; the only question is
          // whether it finished badly.
          status: statusFromResult(isError),
        } satisfies ToolCallContent);
      }
    }

    const failure = (container as { error?: unknown }).error;
    if (failure && typeof failure === "object") {
      const e = failure as Record<string, unknown>;
      if (typeof e.message === "string" && e.message) {
        out.push({
          type: "error",
          message: e.message,
          code: typeof e.code === "string" ? e.code : undefined,
        });
      }
    }

    // `attachments` and `sources` are intentionally not mapped yet — rendering
    // replayed attachments needs a signed-URL endpoint that doesn't exist.
  }
  return out;
}

type RawMessage = {
  id?: string;
  role?: string;
  content?: string;
  metadata?: unknown;
};

function normalizeMessage(raw: RawMessage): Message {
  return {
    id: raw.id,
    role: raw.role === "user" ? "user" : "assistant",
    content: raw.content,
    metadata: flattenMetadata(raw.metadata),
  };
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
  const data = response.data as { messages?: RawMessage[]; has_more?: boolean };
  return {
    messages: (data.messages ?? []).map(normalizeMessage),
    has_more: data.has_more ?? false,
  };
};
