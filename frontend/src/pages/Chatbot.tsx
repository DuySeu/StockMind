import { chatWithLLM, getMessages } from "@/api/chat";
import stockmindLogo from "@/assets/stockmind.png";
import Header from "@/components/containers/Header";
import MessageList from "@/components/containers/MessageList";
import SideBar from "@/components/containers/SideBar";
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
import { SidebarProvider } from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Message } from "@/types/message";
import { ArrowUp, AudioLines, FileText, Image, Paperclip, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

const ChatbotPage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const [title, setTitle] = useState("StockMind");
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
    if (id) {
      setTitle(id);
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
              setMessages((prev) => {
                const updated = [...prev];
                const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
                const existingToolCalls = currentMessage.tool_calls || [];
                updated[assistantIndex] = {
                  ...currentMessage,
                  tool_calls: [...existingToolCalls, tool_calls],
                };
                return updated;
              });
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
      fileToSend,
    );
    if (!id && sessionId) {
      setTitle(data.input.trim());
      navigate(`/${sessionId}`);
    }
  };

  const onHandleSuggestion = (suggestion: string) => {
    form.setValue("input", suggestion);
    onSubmit(form.getValues());
  };

  return (
    <>
      <SidebarProvider className="h-svh bg-sidebar overflow-hidden">
        <SideBar title={title} setTitle={setTitle} />
        <main className="w-full flex flex-col border border-border transition-colors duration-300 rounded-2xl m-3 ml-0 overflow-hidden">
          <div className="flex-1 flex flex-col bg-background-light dark:bg-background-dark relative h-full overflow-hidden">
            <Header shouldAnimate={!!id} title={title} setTitle={setTitle} />
            <div ref={scrollRef} className="flex-1 overflow-hidden flex flex-col relative w-full h-full">
              <ScrollArea className="flex-1 w-full h-full">
                <div className="max-w-4xl mx-auto px-4 py-8 space-y-8 w-full min-h-full flex flex-col">
                  {messages.length > 0 ? (
                    <MessageList messages={messages} />
                  ) : (
                    <div className="flex flex-col items-center justify-center flex-1 text-center gap-4 mt-10">
                      <img src={stockmindLogo} alt="StockMind" className="h-48 w-48 drop-shadow-sm" />
                      <div className="space-y-2 max-w-md">
                        <h2 className="text-2xl font-bold tracking-tight text-primary">Welcome to StockMind</h2>
                        <p className="text-slate-600 dark:text-slate-400">
                          Your AI-powered assistant for market analysis and stock insights. Ask me anything about the
                          stock market!
                        </p>
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 w-full max-w-lg mt-6">
                        {[
                          "What is FPT stock price?",
                          "Should I buy VNM?",
                          "Explain P/E ratio",
                          "get FPT stock price and report",
                        ].map((suggestion, idx) => (
                          <Button key={idx} onClick={() => onHandleSuggestion(suggestion)}>
                            {suggestion}
                          </Button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </ScrollArea>
            </div>

            {/* Fixed Chat Input Area */}
            <div className="absolute bottom-0 left-0 w-full p-4 md:p-6 bg-gradient-to-t from-background dark:from-background-dark via-background/95 dark:via-background-dark/95 to-transparent">
              <div className="max-w-4xl mx-auto w-full relative">
                <Form {...form}>
                  <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col p-2 pl-4 rounded-2xl shadow-xl">
                    {attachment && (
                      <div className="flex items-center gap-2 px-3 py-2 bg-slate-100 dark:bg-slate-700/50 rounded-lg border border-slate-200 dark:border-slate-700 w-fit mb-1 mt-1">
                        <div className="flex items-center gap-2 text-sm">
                          {attachment?.type.startsWith("image/") ? (
                            <Image className="h-4 w-4 text-primary" />
                          ) : (
                            <FileText className="h-4 w-4 text-primary" />
                          )}
                          <span className="max-w-[150px] truncate">{attachment?.name}</span>
                        </div>
                        <button
                          type="button"
                          className="p-1 hover:bg-slate-200 dark:hover:bg-slate-600 rounded-full text-slate-500 transition-colors cursor-pointer"
                          onClick={() => setAttachment(null)}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    )}

                    <div className="flex items-end gap-2 md:gap-4 w-full">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            className="p-2 mb-1 text-slate-400 hover:text-primary transition-colors disabled:opacity-50 cursor-pointer"
                          >
                            <Paperclip className="h-5 w-5" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent className="w-40 rounded-lg border border-primary/20" align="start">
                          <DropdownMenuItem onClick={() => handleFileClick("*/*")} className="cursor-pointer">
                            <FileText className="h-4 w-4 mr-2" />
                            Upload file
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleFileClick("image/*")} className="cursor-pointer">
                            <Image className="h-4 w-4 mr-2" />
                            Upload photo
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>

                      <FormField
                        control={form.control}
                        name="input"
                        render={({ field }) => (
                          <FormItem className="flex-1">
                            <FormControl>
                              <Input
                                className="w-full bg-transparent border-none shadow-none focus-visible:ring-0 placeholder:text-slate-400 py-3 min-h-[44px] text-base resize-none"
                                placeholder="Ask me anything about Vietnam stocks..."
                                autoComplete="off"
                                {...field}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />

                      <div className="flex items-center gap-1 mb-1">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              className="p-2 text-slate-400 hover:text-primary transition-colors hidden md:block cursor-pointer"
                            >
                              <AudioLines className="h-5 w-5" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>Dictate</p>
                          </TooltipContent>
                        </Tooltip>
                        <button
                          type="submit"
                          disabled={!form.watch("input")?.trim()}
                          className="bg-primary hover:bg-primary/90 text-background-dark h-10 w-10 shrink-0 rounded-xl flex items-center justify-center transition-all shadow-md active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
                        >
                          <ArrowUp className="h-5 w-5" />
                          <span className="sr-only">Send</span>
                        </button>
                      </div>
                    </div>
                    <input type="file" ref={fileInputRef} className="hidden" onChange={handleFileChange} />
                  </form>
                </Form>
                <p className="text-center text-[10px] text-slate-400 mt-3 font-medium">
                  StockMind can make mistakes. Verify important financial info.
                </p>
              </div>
            </div>
          </div>
        </main>
      </SidebarProvider>
      <Toaster position="top-right" />
    </>
  );
};

export default ChatbotPage;
