import { chatWithLLM, getMessages } from "@/api/chat";
import { useParams, useNavigate } from "react-router-dom";
import { useChatContext } from "@/hooks/context";
import Header from "@/components/containers/Header";
import MessageList from "@/components/containers/MessageList";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ArrowUp, MessageSquareText, Paperclip } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import type { Message } from "@/types/message";
import stockmindLogo from "@/assets/stockmind.png";

const HomePage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const { title, setTitle } = useChatContext();
  const [messages, setMessages] = useState<Message[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  const form = useForm({
    defaultValues: {
      input: "",
    },
  });

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (scrollRef.current) {
      const scrollElement = scrollRef.current.querySelector('[data-slot="scroll-area-viewport"]');
      if (scrollElement) {
        scrollElement.scrollTop = scrollElement.scrollHeight;
      }
    }
  }, [messages]);

  // Fetch messages when id changes
  useEffect(() => {
    const fetchMessages = async () => {
      if (id) {
        const messages = await getMessages(id);
        setMessages(messages);
      } else {
        setMessages([]);
      }
    };

    fetchMessages();
  }, [id]);

  useEffect(() => {
    if (!id) {
      setTitle("StockMind");
    }
  }, [id]);

  const onSubmit = async (data: FieldValues) => {
    form.reset();
    let sessionId: string | undefined = undefined;

    setMessages((prev) => [...prev, { role: "user", content: [{ type: "text", text: data.input.trim() }] }]);

    const assistantIndex = messages.length + 1;
    setMessages((prev) => [...prev, { role: "assistant", content: [] }]);

    await chatWithLLM(
      data.input.trim(),
      id || undefined,
      (data) => {
        switch (data.type) {
          case "thinking_delta": {
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              const delta = data.data?.thinking ?? "";
              let idx = newContent.findIndex((c) => (c.type as string) === "thinking");
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
            sessionId = data.data?.session_id;
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
    if (!id && sessionId) {
      setTitle(data.input.trim());
      navigate(`/c/${sessionId}`);
    }
  };

  const onHandleSuggestion = (suggestion: string) => {
    form.setValue("input", suggestion);
    onSubmit(form.getValues());
  };

  return (
    <>
      <Header
        icon={<MessageSquareText className="text-primary w-6 h-6" />}
        editable={title !== "StockMind"}
        shouldAnimate={!!id}
      />
      <div ref={scrollRef} className="flex-1 overflow-hidden flex flex-col w-full min-h-0">
        <ScrollArea className="flex-1 w-full min-h-0 px-6">
          {messages.length > 0 ? (
            <MessageList messages={messages} />
          ) : (
            <div className="flex flex-col items-center justify-center min-h-[50vh] text-center gap-4">
              <img src={stockmindLogo} alt="StockMind" className="h-48 w-48" />
              <div className="space-y-2 max-w-md">
                <h2 className="text-2xl font-bold tracking-tight text-primary">Welcome to StockMind</h2>
                <p className="text-muted-foreground">
                  Your AI-powered assistant for market analysis and stock insights. Ask me anything about the stock
                  market!
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-2 w-full max-w-lg mt-4">
                {["What is FPT stock price?", "Should I buy VNM?", "Explain P/E ratio", "Latest tech news"].map(
                  (suggestion, idx) => (
                    <button
                      key={idx}
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
        </ScrollArea>

        <div className="my-2 w-full flex justify-center pointer-events-none shrink-0">
          <div className="w-full max-w-3xl pointer-events-auto shadow-2xl rounded-2xl bg-background/80 backdrop-blur-md border border-border/50">
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="p-2">
                <FormField
                  control={form.control}
                  name="input"
                  render={({ field }) => (
                    <FormItem className="flex-1">
                      <FormControl>
                        <Input
                          className="border-0 focus-visible:ring-0 focus-visible:ring-offset-0 bg-transparent py-6 text-base text-primary shadow-none resize-none"
                          placeholder="Ask StockMind about market trends..."
                          autoComplete="off"
                          {...field}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <Button type="button" variant="secondary" size="icon-sm" className="bg-background/80">
                      <Paperclip />
                    </Button>
                  </div>
                  <Button
                    type="submit"
                    size="icon-sm"
                    className="bg-primary hover:bg-primary/90 transition-all shadow-sm mr-1"
                    disabled={!form.watch("input")?.trim()}
                  >
                    <ArrowUp className="h-5 w-5" />
                    <span className="sr-only">Send</span>
                  </Button>
                </div>
              </form>
            </Form>
          </div>
        </div>
      </div>
    </>
  );
};

export default HomePage;
