import { chatWithLLM, getMessages } from "@/api/chat";
import stockmindLogo from "@/assets/stockmind.png";
import Header from "@/components/containers/Header";
import MessageList from "@/components/containers/MessageList";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChatContext } from "@/hooks/context";
import type { Message } from "@/types/message";
import { ArrowUp, AudioLines, FileText, Image, MessageSquareText, Paperclip, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

const HomePage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const { title, setTitle } = useChatContext();
  const [messages, setMessages] = useState<Message[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [attachment, setAttachment] = useState<File | null>(null);

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

  const handleFileClick = (accept: string) => {
    if (fileInputRef.current) {
      fileInputRef.current.accept = accept;
      fileInputRef.current.click();
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      // 10MB limit
      if (file.size > 10 * 1024 * 1024) {
        toast.error("File is too large. Max size is 10MB.");
      } else {
        setAttachment(file);
      }
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const onSubmit = async (data: FieldValues) => {
    const fileToSend = attachment; // Capture attachment before clearing
    form.reset();
    setAttachment(null);
    let sessionId: string | undefined = undefined;

    const content: any[] = [];
    if (fileToSend) {
      content.push({
        type: "image_url",
        image_url: {
          url: URL.createObjectURL(fileToSend),
        },
      });
    }
    content.push({ type: "text", text: data.input.trim() });

    setMessages((prev) => [
      ...prev,
      {
        role: "user",
        content: content,
      },
    ]);

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
              let idx = newContent.findIndex((c) => c.type === "thinking");
              if (idx === -1) {
                newContent.push({
                  type: "thinking",
                  thinking: "",
                  is_open: true,
                });
                idx = newContent.length - 1;
              }
              const block = newContent[idx];
              if (block.type === "thinking") {
                newContent[idx] = { ...block, thinking: (block.thinking ?? "") + delta };
              }
              updated[assistantIndex] = { ...currentMessage, content: newContent };
              return updated;
            });
            break;
          }
          case "tool_use": {
            const tool_calls = data.data?.tool_calls;
            if (tool_calls) {
              setMessages((prev) => [
                ...prev,
                {
                  role: "assistant",
                  tool_calls: [tool_calls],
                },
              ]);
            }
            break;
          }
          case "tool_result": {
            setMessages((prev) => [...prev, data.data?.result as Message]);
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
              if (block.type === "text") {
                newContent[idx] = { ...block, text: (block.text ?? "") + delta };
              }
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
                if (block.type === "thinking") {
                  newContent[idx] = { ...block, is_open: false };
                }
              }
              updated[assistantIndex] = { ...currentMessage, content: newContent };
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
      },
      fileToSend
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
                {attachment && (
                  <div className="relative flex items-center gap-2 px-3 py-2 bg-muted/50 rounded-md border border-border/50 w-fit mb-2 ml-2 mt-2">
                    <div className="flex items-center gap-2 text-sm text-foreground/80">
                      {attachment.type.startsWith("image/") ? (
                        <Image className="h-4 w-4" />
                      ) : (
                        <FileText className="h-4 w-4" />
                      )}
                      <span className="max-w-[150px] truncate">{attachment.name}</span>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className="h-5 w-5 rounded-full hover:bg-background/80"
                      onClick={() => setAttachment(null)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                )}
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
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        type="button"
                        variant="secondary"
                        size="icon-sm"
                        className="bg-background/80 data-[state=open]:bg-secondary data-[state=open]:text-secondary-foreground"
                      >
                        <Paperclip />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      className="w-(--radix-dropdown-menu-trigger-width) min-w-40 rounded-lg border border-border"
                      align="start"
                    >
                      <DropdownMenuItem onClick={() => handleFileClick("*/*")}>
                        <FileText className="h-4 w-4" />
                        Upload file
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleFileClick("image/*")}>
                        <Image className="h-4 w-4" />
                        Upload photo
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <div className="flex items-center gap-2">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          variant="secondary"
                          size="icon-sm"
                          className="bg-background/80 rounded-full"
                        >
                          <AudioLines className="h-5 w-5" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Dictate</p>
                      </TooltipContent>
                    </Tooltip>
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
                </div>
                <input type="file" ref={fileInputRef} className="hidden" onChange={handleFileChange} />
              </form>
            </Form>
          </div>
        </div>
      </div>
    </>
  );
};

export default HomePage;
