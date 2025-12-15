import { Bot, Brain, User } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const MessageList = ({ messages }: { messages: Message[] }) => {
  return messages.map((message, index) => {
    const isUser = message.role === "user";
    return (
      <div key={index} className={`flex gap-4 ${isUser ? "flex-row-reverse" : "flex-row"} items-end`}>
        {/* Avatar */}
        <div
          className={`flex-shrink-0 h-10 w-10 rounded-full flex items-center justify-center ${
            isUser ? "bg-primary text-primary-foreground" : "bg-accent text-accent-foreground border border-border"
          }`}
        >
          {isUser ? <User className="h-6 w-6" /> : <Bot className="h-6 w-6" />}
        </div>

        {/* Message Content */}
        <div className={`flex flex-col max-w-[85%] ${isUser ? "items-end" : "items-start"}`}>
          {/* Name Label (Optional, maybe skip for cleaner look or just show role) */}
          {/* <span className="text-xs text-muted-foreground mb-1 px-1">{isUser ? "You" : "StockMind"}</span> */}

          <div
            className={`px-4 py-2 shadow-sm text-sm md:text-base leading-relaxed ${
              isUser
                ? "rounded-2xl rounded-br-sm bg-primary text-primary-foreground"
                : "rounded-2xl rounded-bl-sm bg-card border border-border text-card-foreground"
            }`}
          >
            {isUser && message.content[0].type === "text" ? (
              <Markdown remarkPlugins={[remarkGfm]}>{message.content[0].text as string}</Markdown>
            ) : (
              message.content.map((content, idx) => {
                switch (content.type) {
                  case "text":
                    return (
                      <Markdown key={idx} remarkPlugins={[remarkGfm]}>
                        {content.text as string}
                      </Markdown>
                    );
                  case "thinking":
                    return (
                      <div key={idx} className="mb-2">
                        <details
                          open={content.is_open as boolean}
                          className="group border border-border/50 rounded-lg bg-muted/30 overflow-hidden"
                        >
                          <summary className="flex items-center gap-2 p-2 text-xs font-medium select-none cursor-pointer text-muted-foreground hover:bg-muted/50 transition-colors">
                            <Brain className="h-3 w-3" />
                            <span>Thought Process</span>
                          </summary>
                          <div className="p-3 pt-0 text-xs text-muted-foreground/80 leading-relaxed border-t border-border/20">
                            <Markdown remarkPlugins={[remarkGfm]}>{content.thinking as string}</Markdown>
                          </div>
                        </details>
                      </div>
                    );
                  default:
                    return null;
                }
              })
            )}
          </div>
        </div>
      </div>
    );
  });
};

export default MessageList;
