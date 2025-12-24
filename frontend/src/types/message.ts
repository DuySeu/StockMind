export type TextContent = {
  type: "text";
  text: string;
};

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

export type ContentPart = TextContent | ImageContent | ThinkingContent;

export type Message = {
  role: "user" | "assistant" | "tool";
  content?: ContentPart[];
  tool_calls?: Record<string, unknown>[];
};
