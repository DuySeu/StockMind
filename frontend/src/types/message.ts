export type ImageContent = {
  type: "image_url";
  image_url: {
    url: string;
  };
};

export type ThinkingContent = {
  type: "thinking";
  thinking: string;
  is_open: boolean;
};

export type ToolCallContent = {
  type: "tool_call";
  id: string;
  name: string;
  arguments?: string;
  result?: string;
  is_error?: boolean;
  /** `running` until a matching `tool_result` arrives (or is replayed). */
  status: "running" | "success" | "error";
};

/** @deprecated Kept as an alias so existing imports keep compiling. */
export type ToolCall = ToolCallContent;

/**
 * Why an assistant turn failed. Carried both by the in-stream `error` event and
 * by the turn's persisted metadata, so a failed turn survives a reload instead
 * of leaving the user's question sitting there with no reply.
 */
export type TurnErrorContent = {
  type: "error";
  message: string;
  code?: string;
};

/**
 * The server's code for an upstream quota or rate-limit refusal
 * (`failQuota` in internal/server/chat.handler.go). It reads differently from
 * every other failure: nothing is broken, retrying now changes nothing, and only
 * the account owner can act on it — so it is shown in the thread and never as a
 * toast.
 */
export const QUOTA_ERROR_CODE = "quota_exhausted";

export const isQuotaError = (code?: string): boolean => code === QUOTA_ERROR_CODE;

export type Metadata = ImageContent | ThinkingContent | ToolCallContent | TurnErrorContent;

export type Message = {
  id?: string;
  role: "user" | "assistant";
  content?: string;
  metadata?: Metadata[];
};

/** Narrowing helper — every renderer should go through this, never a structural check. */
export const isToolCall = (m: Metadata): m is ToolCallContent => m.type === "tool_call";

export const isThinking = (m: Metadata): m is ThinkingContent => m.type === "thinking";

export const isImage = (m: Metadata): m is ImageContent => m.type === "image_url";

export const isTurnError = (m: Metadata): m is TurnErrorContent => m.type === "error";

/** The wire sends `is_error` as the string `"true"`; accept both spellings. */
export const toIsError = (raw: unknown): boolean => raw === true || raw === "true";

/** Status a tool sits in once its result is known. */
export const statusFromResult = (isError: boolean): ToolCallContent["status"] => (isError ? "error" : "success");
