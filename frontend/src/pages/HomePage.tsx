import { chatWithLLM } from "@/api/chat";
import MessageList from "@/components/containers/MessageList";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Bot, Moon, Send, Sun } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";

type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const HomePage = () => {
  const [conversationId] = useState<string | null>(null);
  const [isFirstMessage, setIsFirstMessage] = useState<boolean>(true);
  const [messages, setMessages] = useState<Message[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [theme, setTheme] = useState<"light" | "dark">("light");

  const form = useForm({
    defaultValues: {
      input: "",
    },
  });

  useEffect(() => {
    // Check system theme preference
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      setTheme("dark");
      document.documentElement.classList.add("dark");
    }
  }, []);

  const toggleTheme = () => {
    if (theme === "light") {
      setTheme("dark");
      document.documentElement.classList.add("dark");
    } else {
      setTheme("light");
      document.documentElement.classList.remove("dark");
    }
  };

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (scrollRef.current) {
      const scrollElement = scrollRef.current.querySelector("[data-radix-scroll-area-viewport]");
      if (scrollElement) {
        scrollElement.scrollTop = scrollElement.scrollHeight;
      }
    }
  }, [messages]);

  const onSubmit = async (data: FieldValues) => {
    form.reset();

    if (!conversationId) {
      setIsFirstMessage(true);
    }

    setMessages((prev) => [...prev, { role: "user", content: [{ type: "text", text: data.input.trim() }] }]);

    const assistantIndex = messages.length + 1;
    setMessages((prev) => [...prev, { role: "assistant", content: [] }]);

    await chatWithLLM(
      data.input.trim(),
      conversationId || undefined,
      (data) => {
        switch (data.type) {
          case "thinking_delta": {
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              const delta = data.data?.thinking ?? "";
              let idx = newContent.findIndex((c) => c.type === "thinking");
              if (idx === -1) {
                newContent.push({
                  type: "thinking",
                  thinking: "",
                  signature: "",
                  is_open: true,
                });
                idx = newContent.length - 1;
              }
              const block = newContent[idx];
              newContent[idx] = { ...block, thinking: (block.thinking ?? "") + delta };
              updated[assistantIndex] = { ...currentMessage, content: newContent };
              return updated;
            });
            break;
          }
          case "text_delta": {
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              const delta = data.data?.text ?? "";
              let idx = newContent.findIndex((c) => c.type === "text");
              if (idx === -1) {
                newContent.push({ type: "text", text: "" });
                idx = newContent.length - 1;
              }
              const block = newContent[idx];
              newContent[idx] = { ...block, text: (block.text ?? "") + delta };
              updated[assistantIndex] = { ...currentMessage, content: newContent };
              return updated;
            });
            break;
          }
          case "complete": {
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              const idx = newContent.findIndex((c) => c.type === "thinking");
              if (idx !== -1) {
                const block = newContent[idx];
                newContent[idx] = { ...block, is_open: false };
                updated[assistantIndex] = { ...currentMessage, content: newContent };
              }
              return updated;
            });
            break;
          }
        }
      },
      (error) => {
        console.error("Error sending message:", error);
        setMessages((prev) => {
          const updated = [...prev];
          updated[assistantIndex] = {
            role: "assistant",
            content: [{ type: "text", text: "Error sending message" }],
          };
          return updated;
        });
      }
    );

    if (isFirstMessage) {
      setIsFirstMessage(false);
    }
  };

  const onHandleSuggestion = (suggestion: string) => {
    form.setValue("input", suggestion);
    onSubmit(form.getValues());
  };

  return (
    <>
      <header className="flex justify-between items-center py-3 px-4">
        <div className="flex items-center gap-2">
          {/* <SidebarTrigger className="text-primary" /> */}
          <span className="font-semibold text-lg text-primary hidden md:block">StockMind AI</span>
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleTheme}
          className="rounded-full hover:bg-muted text-primary"
          aria-label="Toggle Theme"
        >
          {theme === "light" ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
        </Button>
      </header>
      <div className="flex-1 overflow-hidden flex flex-col relative w-full max-w-4xl mx-auto">
        <ScrollArea ref={scrollRef} className="flex-1 p-4 md:p-6 w-full">
          <div className="pb-24 max-w-3xl mx-auto">
            <div className="flex-1 flex flex-col gap-6">
              {messages.length > 0 ? (
                <MessageList messages={messages} />
              ) : (
                <div className="flex flex-col items-center justify-center min-h-[50vh] text-center gap-4">
                  <div className="bg-primary/5 rounded-full p-6 ring-1 ring-primary/10">
                    <Bot className="h-12 w-12 text-primary" />
                  </div>
                  <div className="space-y-2 max-w-md">
                    <h2 className="text-2xl font-bold tracking-tight text-primary">Welcome to StockMind</h2>
                    <p className="text-muted-foreground">
                      Your AI-powered assistant for market analysis and stock insights. Ask me anything about the stock
                      market!
                    </p>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2 w-full max-w-lg mt-4">
                    {["Analyze market trends", "Should I buy AAPL?", "Explain P/E ratio", "Latest tech news"].map(
                      (suggestion) => (
                        <button
                          key={suggestion}
                          onClick={() => onHandleSuggestion(suggestion)}
                          className="p-3 text-sm text-left border rounded-xl hover:bg-accent transition-colors text-muted-foreground hover:text-foreground"
                        >
                          {suggestion}
                        </button>
                      )
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </ScrollArea>

        <div className="absolute bottom-6 left-0 right-0 px-4 w-full flex justify-center z-20 pointer-events-none">
          <div className="w-full max-w-3xl pointer-events-auto shadow-2xl rounded-2xl bg-background/80 backdrop-blur-md border border-border/50">
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="flex items-center gap-2 p-2 relative">
                <FormField
                  control={form.control}
                  name="input"
                  render={({ field }) => (
                    <FormItem className="flex-1">
                      <FormControl>
                        <Input
                          className="border-0 focus-visible:ring-0 focus-visible:ring-offset-0 bg-transparent py-6 pl-4 text-base text-primary shadow-none resize-none"
                          placeholder="Ask StockMind about market trends..."
                          autoComplete="off"
                          {...field}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <Button
                  type="submit"
                  size="icon"
                  className="h-10 w-10 shrink-0 rounded-xl bg-primary hover:bg-primary/90 transition-all shadow-sm mr-1"
                  disabled={!form.watch("input")?.trim()}
                >
                  <Send className="h-5 w-5" />
                  <span className="sr-only">Send</span>
                </Button>
              </form>
            </Form>
          </div>
        </div>
      </div>
    </>
  );
};

export default HomePage;
