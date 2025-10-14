type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const MessageList = ({ messages }: { messages: Message[] }) => {
  return (
    <div className="flex-1 gap-2">
      {messages.length > 0 ? (
        messages.map((message, index) => {
          if (message.role === "user" && message.content[0].type === "text") {
            return (
              <div
                key={index}
                className="self-end max-w-[85%] rounded-2xl bg-primary text-primary-foreground px-3 py-2 shadow-sm m-3"
              >
                <div>{message.content[0].text as string}</div>
              </div>
            );
          } else if (message.role === "assistant") {
            return (
              <div key={index} className="self-start max-w-[85%] rounded-2xl text-secondary-foreground px-3 py-2">
                {message.content.map((content, index) => {
                  switch (content.type) {
                    case "text":
                      return <div key={index}>{content.text as string}</div>;
                    case "thinking":
                      return (
                        <details key={index} open={content.is_open as boolean} className="group overflow-hidden">
                          <summary className="p-2 text-xs font-medium select-none cursor-pointer text-secondary-foreground/50">
                            Thinking
                          </summary>
                          <div className="p-2 leading-5 text-secondary-foreground/50 whitespace-pre-wrap">
                            {content.thinking as string}
                          </div>
                        </details>
                      );
                  }
                })}
              </div>
            );
          }
        })
      ) : (
        <div className="flex items-center justify-center h-full">
          <div className="overflow-hidden whitespace-nowrap text-4xl text-center text-primary/50 select-none">
            Hello, I'm StockMind. How can I help you?
          </div>
        </div>
      )}
    </div>
  );
};

export default MessageList;
