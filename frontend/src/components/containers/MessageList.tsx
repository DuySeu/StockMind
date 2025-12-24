import type { Message } from "@/types/message";
import { Brain } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

const MessageList = ({ messages }: { messages: Message[] }) => {
  return messages
    .filter((message) => message.role !== "tool" && !message.tool_calls)
    .map((message, index) => {
      const isUser = message.role === "user";
      return (
        <div key={index} className={`flex gap-4 ${isUser ? "flex-row-reverse" : "flex-row"} items-end`}>
          {/* Message Content */}
          <div className={`flex flex-col max-w-[85%] ${isUser ? "items-end" : "items-start"}`}>
            {message.content && message.content.length === 0 && (
              <span className="text-xs text-muted-foreground mb-1 px-1">Thinking</span>
            )}
            {message.content && message.content.length > 0 ? (
              [...message.content]
                .sort((a, b) => {
                  if (a.type === "image_url" && b.type !== "image_url") return -1;
                  if (a.type !== "image_url" && b.type === "image_url") return 1;
                  return 0;
                })
                .map((content, idx) => {
                  if (content.type === "image_url") {
                    return (
                      <div key={idx} className="mb-2 max-w-full">
                        <img
                          src={content.image_url?.url}
                          alt="Attachment"
                          className="max-w-full h-auto rounded-lg border border-border shadow-sm max-h-64 object-contain inline-block bg-background"
                        />
                      </div>
                    );
                  }

                  // Text and Thinking parts get the bubble style
                  return (
                    <div
                      key={idx}
                      className={`px-4 py-2 shadow-sm text-sm md:text-base leading-relaxed mb-1 ${
                        isUser ? "rounded-2xl rounded-tr-sm bg-primary text-primary-foreground" : "text-card-foreground"
                      }`}
                    >
                      {content.type === "text" && <Markdown remarkPlugins={[remarkGfm]}>{content.text}</Markdown>}
                      {content.type === "thinking" && (
                        <div className="mb-2">
                          <details
                            open={content.is_open}
                            className="group border border-border/50 rounded-lg overflow-hidden"
                          >
                            <summary className="flex items-center gap-2 p-2 text-xs font-medium select-none cursor-pointer text-muted-foreground hover:bg-muted/50 transition-colors">
                              <Brain className="h-3 w-3" />
                              <span>Thought Process</span>
                            </summary>
                            <div className="p-3 pt-0 text-xs text-muted-foreground/80 leading-relaxed border-t border-border/20">
                              <Markdown remarkPlugins={[remarkGfm]}>{content.thinking}</Markdown>
                            </div>
                          </details>
                        </div>
                      )}
                    </div>
                  );
                })
            ) : (
              <div
                className={`px-4 py-2 shadow-sm text-sm md:text-base leading-relaxed ${
                  isUser ? "rounded-2xl rounded-tr-sm bg-primary text-primary-foreground" : "text-card-foreground"
                }`}
              >
                <div className="flex items-center space-x-1">
                  <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-0"></span>
                  <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-150"></span>
                  <span className="h-2 w-2 animate-pulse rounded-full bg-primary delay-300"></span>
                </div>
              </div>
            )}
          </div>
        </div>
      );
    });
};

export default MessageList;
