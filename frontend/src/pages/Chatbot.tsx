import { sendChatMessage, getMessages } from "@/api/chat";
import type { ChatEvent } from "@/api/chat";
import stockmindLogo from "@/assets/stockmind.png";
import Header from "@/components/containers/Header";
import MessageList from "@/components/containers/MessageList";
import SideBar from "@/components/containers/SideBar";
import { Navbar } from "@/components/layout/Navbar";
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
import type { Message, ToolCall } from "@/types/message";
import { ArrowUp, AudioLines, FileText, Image, Paperclip, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
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
  const [sessionVersion, setSessionVersion] = useState(0);
  const [isStreaming, setIsStreaming] = useState(false);

  const streamingSessionRef = useRef<string | null>(null);

  const refreshSessions = useCallback(() => setSessionVersion((v) => v + 1), []);

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

  // Fetch messages when navigating to an existing session.
  // Skip if onSubmit is already streaming this session (streamingSessionRef).
  useEffect(() => {
    if (!id) {
      setMessages([]);
      return;
    }

    // onSubmit already owns the stream for this session — don't duplicate.
    if (streamingSessionRef.current === id) return;

    let cancelled = false;
    const fetchMessages = async () => {
      const loadedMessages = await getMessages(id);
      if (cancelled) return;
      setMessages(loadedMessages);
    };

    fetchMessages();
    return () => {
      cancelled = true;
    };
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

  /**
   * Handle a single SSE event from the backend and update messages state.
   * Returns the session_id when a "start" event provides one.
   */
  const handleStreamEvent = useCallback((event: ChatEvent, callbacks: { onSessionId?: (sid: string) => void }) => {
    const { type, data } = event;

    switch (type) {
      case "start": {
        const sid = (data as { session_id?: string })?.session_id;
        if (sid) callbacks.onSessionId?.(sid);
        break;
      }

      case "text": {
        const delta = typeof data === "string" ? data : "";
        setMessages((prev) => {
          const updated = [...prev];
          const lastIndex = updated.length - 1;

          if (lastIndex < 0 || updated[lastIndex].role !== "assistant") {
            updated.push({ role: "assistant", content: delta });
            return updated;
          }

          const current = updated[lastIndex];
          updated[lastIndex] = {
            ...current,
            content: (current.content ?? "") + delta,
          };
          return updated;
        });
        break;
      }

      case "thinking": {
        const delta = typeof data === "string" ? data : "";
        setMessages((prev) => {
          const updated = [...prev];
          const lastIndex = updated.length - 1;

          if (lastIndex < 0 || updated[lastIndex].role !== "assistant") {
            updated.push({
              role: "assistant",
              content: "",
              metadata: [{ type: "thinking", thinking: delta, is_open: true }],
            });
            return updated;
          }

          const current = updated[lastIndex];
          const existingMeta = current.metadata ?? [];
          const thinkingIdx = existingMeta.findIndex((m) => "thinking" in m && m.type === "thinking");

          if (thinkingIdx === -1) {
            updated[lastIndex] = {
              ...current,
              metadata: [...existingMeta, { type: "thinking", thinking: delta, is_open: true }],
            };
          } else {
            const newMeta = [...existingMeta];
            const existing = newMeta[thinkingIdx] as { type: "thinking"; thinking: string; is_open: boolean };
            newMeta[thinkingIdx] = {
              ...existing,
              thinking: existing.thinking + delta,
            };
            updated[lastIndex] = { ...current, metadata: newMeta };
          }
          return updated;
        });
        break;
      }

      case "tool_call": {
        if (data && typeof data === "object") {
          setMessages((prev) => {
            const updated = [...prev];
            const lastIndex = updated.length - 1;

            if (lastIndex < 0 || updated[lastIndex].role !== "assistant") {
              updated.push({
                role: "assistant",
                content: "",
                metadata: [data as ToolCall],
              });
              return updated;
            }

            const current = updated[lastIndex];
            updated[lastIndex] = {
              ...current,
              metadata: [...(current.metadata ?? []), data as ToolCall],
            };
            return updated;
          });
        }
        break;
      }

      case "tool_result": {
        // Tool results can be appended as metadata or as a separate message
        // depending on how MessageList renders them.
        // For now, we skip tool_result display (tool output is usually
        // digested by the LLM into the next text response).
        break;
      }

      case "done": {
        // Close any open thinking blocks
        setMessages((prev) => {
          const updated = [...prev];
          const lastIndex = updated.length - 1;
          if (lastIndex >= 0 && updated[lastIndex].role === "assistant") {
            const current = updated[lastIndex];
            if (current.metadata) {
              const newMeta = current.metadata.map((m) => {
                if ("thinking" in m && m.type === "thinking") {
                  return { ...m, is_open: false };
                }
                return m;
              });
              updated[lastIndex] = { ...current, metadata: newMeta };
            }
          }
          return updated;
        });
        break;
      }

      case "error": {
        const message = typeof data === "string" ? data : ((data as { message?: string })?.message ?? "Stream error");
        toast.error(message);
        break;
      }
    }
  }, []);

  const onSubmit = async (data: FieldValues) => {
    const userInput = data.input.trim();
    if (!userInput) return;

    const fileToSend = attachment; // Capture attachment before clearing
    form.reset();
    setAttachment(null);
    setIsStreaming(true);

    // Add user message to the list
    setMessages((prev) => [
      ...prev,
      {
        role: "user",
        content: userInput,
        metadata: fileToSend
          ? [
              {
                type: "image_url" as const,
                image_url: { url: URL.createObjectURL(fileToSend) },
              },
            ]
          : [],
      },
    ]);

    try {
      await sendChatMessage(
        userInput,
        id || undefined,
        (event) => {
          handleStreamEvent(event, {
            onSessionId: (sid) => {
              streamingSessionRef.current = sid;

              // Navigate to the new session URL if this is a fresh chat
              if (!id && sid) {
                setTitle(userInput);
                navigate(`/c/${sid}`, { replace: true });
                refreshSessions();
              }
            },
          });
        },
        (error) => {
          console.error("Stream error:", error);
          setMessages((prev) => [
            ...prev,
            { role: "assistant", content: "An error occurred while processing your request." },
          ]);
        },
        fileToSend,
      );

      // Stream finished — release ownership so future navigations can reconnect.
      streamingSessionRef.current = null;
      setIsStreaming(false);
    } catch (err) {
      streamingSessionRef.current = null;
      setIsStreaming(false);
      console.error(err);
    }
  };

  const onHandleSuggestion = (suggestion: string) => {
    form.setValue("input", suggestion);
    onSubmit(form.getValues());
  };

  return (
    <>
      <SidebarProvider className="flex-1 w-full h-screen bg-sidebar overflow-hidden">
        <SideBar title={title} setTitle={setTitle} sessionVersion={sessionVersion} />
        <main className="w-full h-full flex flex-col">
          <div className="mt-4 px-2">
            <Navbar />
          </div>
          <div className="flex-1 flex flex-col bg-background-light dark:bg-background-dark relative h-full border border-border transition-colors duration-300 rounded-2xl m-4 ml-0 overflow-hidden">
            <Header shouldAnimate={!!id} title={title} setTitle={setTitle} />
            <div ref={scrollRef} className="flex-1 overflow-hidden flex flex-col relative w-full h-full">
              <ScrollArea className="flex-1 w-full h-full">
                <div className="max-w-4xl mx-auto px-4 pt-8 pb-36 md:pb-40 space-y-8 w-full min-h-full flex flex-col">
                  {messages.length > 0 || id ? (
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
                                disabled={isStreaming}
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
                          disabled={isStreaming || !form.watch("input")?.trim()}
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
