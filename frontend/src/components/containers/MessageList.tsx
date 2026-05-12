import type { Message, ThinkingContent, ToolCall, ImageContent } from "@/types/message";
import { Brain, Check, LoaderCircle, Wrench } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

const markdownComponents = {
  table: (props: React.ComponentProps<"table">) => (
    <div className="my-3 overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm border-collapse" {...props} />
    </div>
  ),
  thead: (props: React.ComponentProps<"thead">) => (
    <thead className="bg-muted/60 text-left" {...props} />
  ),
  th: (props: React.ComponentProps<"th">) => (
    <th className="px-3 py-2 font-semibold border-b border-border whitespace-nowrap" {...props} />
  ),
  td: (props: React.ComponentProps<"td">) => (
    <td className="px-3 py-1.5 border-b border-border/50 tabular-nums" {...props} />
  ),
  tr: (props: React.ComponentProps<"tr">) => (
    <tr className="even:bg-muted/30" {...props} />
  ),
  code: ({ className, children, ...props }: React.ComponentProps<"code"> & { inline?: boolean }) => {
    const isBlock = className?.startsWith("language-");
    if (isBlock) {
      return (
        <pre className="my-2 overflow-x-auto rounded-lg bg-muted/60 p-3 text-xs leading-relaxed">
          <code className={className} {...props}>{children}</code>
        </pre>
      );
    }
    return (
      <code className="rounded bg-muted/60 px-1.5 py-0.5 text-[0.85em] font-mono" {...props}>
        {children}
      </code>
    );
  },
  ul: (props: React.ComponentProps<"ul">) => (
    <ul className="my-1 ml-4 list-disc space-y-0.5" {...props} />
  ),
  ol: (props: React.ComponentProps<"ol">) => (
    <ol className="my-1 ml-4 list-decimal space-y-0.5" {...props} />
  ),
  p: (props: React.ComponentProps<"p">) => (
    <p className="my-1.5 leading-relaxed" {...props} />
  ),
  strong: (props: React.ComponentProps<"strong">) => (
    <strong className="font-semibold" {...props} />
  ),
  h1: (props: React.ComponentProps<"h1">) => <h1 className="text-lg font-bold mt-4 mb-2" {...props} />,
  h2: (props: React.ComponentProps<"h2">) => <h2 className="text-base font-bold mt-3 mb-1.5" {...props} />,
  h3: (props: React.ComponentProps<"h3">) => <h3 className="text-sm font-bold mt-2 mb-1" {...props} />,
};

const MessageList = ({ messages }: { messages: Message[] }) => {
  return messages.map((message, index) => {
    const isUser = message.role === "user";

    // Extract metadata parts
    const images = (message.metadata?.filter((m) => "type" in m && m.type === "image_url") ?? []) as ImageContent[];
    const thinkingBlocks = (message.metadata?.filter(
      (m) => "type" in m && m.type === "thinking",
    ) ?? []) as ThinkingContent[];
    const toolCalls = (message.metadata?.filter(
      (m) => "name" in m && "id" in m,
    ) ?? []) as ToolCall[];

    const hasToolCalls = toolCalls.length > 0;
    // Tool is "in progress" if there's no content yet after tool calls
    const isUsingTool = hasToolCalls && !message.content;

    return (
      <div key={index} className={`flex gap-4 ${isUser ? "flex-row-reverse" : "flex-row"} items-end`}>
        <div className={`flex flex-col max-w-[85%] ${isUser ? "items-end" : "items-start"}`}>
          {/* Image attachments (user messages) */}
          {images.map((img, idx) => (
            <div key={`img-${idx}`} className="mb-2 max-w-full">
              <img
                src={img.image_url?.url}
                alt="Attachment"
                className="max-w-full h-auto rounded-lg border border-border shadow-sm max-h-64 object-contain inline-block bg-background"
              />
            </div>
          ))}

          {/* Thinking blocks (assistant messages) */}
          {thinkingBlocks.map((block, idx) => (
            <div key={`think-${idx}`} className="mb-2 w-full">
              <details
                open={block.is_open}
                className="group border border-border/50 rounded-lg overflow-hidden"
              >
                <summary className="flex items-center gap-2 p-2 text-xs font-medium select-none cursor-pointer text-muted-foreground hover:bg-muted/50 transition-colors">
                  <Brain className="h-3 w-3" />
                  <span>Thought Process</span>
                </summary>
                <div className="p-3 pt-0 text-xs text-muted-foreground/80 leading-relaxed border-t border-border/20">
                  <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
                    {block.thinking}
                  </Markdown>
                </div>
              </details>
            </div>
          ))}

          {/* Tool call indicator */}
          {hasToolCalls && (
            <span
              className={`text-xs px-1 flex items-center gap-1 mb-1 ${
                isUsingTool ? "text-muted-foreground" : "text-accent"
              }`}
            >
              {isUsingTool ? (
                <>
                  <LoaderCircle className="h-3 w-3 animate-spin" />
                  Using tool...
                </>
              ) : (
                <>
                  <Check className="h-4 w-4" />
                  Tool used
                </>
              )}
            </span>
          )}

          {/* Tool call details */}
          {toolCalls.map((tc, idx) => (
            <div
              key={`tc-${idx}`}
              className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1 px-1"
            >
              <Wrench className="h-3 w-3" />
              <span className="font-mono">{tc.name}</span>
            </div>
          ))}

          {/* Main text content */}
          {message.content ? (
            <div
              className={`px-4 py-2 text-sm md:text-base leading-relaxed ${
                isUser
                  ? "rounded-2xl rounded-tr-sm shadow-sm bg-primary text-primary-foreground"
                  : "text-card-foreground"
              }`}
            >
              <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
                {message.content}
              </Markdown>
            </div>
          ) : (
            // Show loading indicator for empty assistant messages (still streaming)
            !isUser && (
              <>
                <span className="text-xs text-muted-foreground mb-1 px-1">Thinking</span>
                <div className="px-4 py-2 text-sm md:text-base leading-relaxed text-card-foreground">
                  <div className="flex items-center space-x-1">
                    <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-0"></span>
                    <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-150"></span>
                    <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-300"></span>
                  </div>
                </div>
              </>
            )
          )}
        </div>
      </div>
    );
  });
};

export default MessageList;
