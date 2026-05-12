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

export type ToolCall = {
  id: string;
  name: string;
  arguments: string;
  output: string;
  is_error: string;
};

export type Metadata = ImageContent | ThinkingContent | ToolCall;

export type Message = {
  role: "user" | "assistant";
  content?: string;
  metadata?: Metadata[];
};
