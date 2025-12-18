export type Message = {
  role: "user" | "assistant" | "tool";
  content: Record<string, unknown>[];
  tool_calls?: Record<string, unknown>[];
};
