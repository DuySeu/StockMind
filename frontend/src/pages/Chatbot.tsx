import { sendChatMessage } from "@/api/chat";
import type { ChatEvent } from "@/api/chat";
import { getSessionMessages } from "@/api/sessions";
import stockmindLogo from "@/assets/stockmind.png";
import ChatInput from "@/components/containers/ChatInput";
import Header from "@/components/containers/Header";
import MessageList from "@/components/containers/MessageList";
import SideBar from "@/components/containers/SideBar";
import { Navbar } from "@/components/layout/Navbar";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { SidebarProvider } from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import type { Message, ToolCall } from "@/types/message";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

const PAGE_SIZE = 20;

const ChatbotPage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const [title, setTitle] = useState("StockMind");
  const [messages, setMessages] = useState<Message[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [sessionVersion, setSessionVersion] = useState(0);
  const [isStreaming, setIsStreaming] = useState(false);
  const streamingSessionRef = useRef<string | null>(null);

  const refreshSessions = useCallback(() => setSessionVersion((v) => v + 1), []);

  // Auto-scroll to bottom when messages change (only when not loading older messages)
  useEffect(() => {
    if (isLoadingMore) return;
    if (scrollRef.current) {
      const scrollElement = scrollRef.current.querySelector('[data-slot="scroll-area-viewport"]');
      if (scrollElement) {
        scrollElement.scrollTop = scrollElement.scrollHeight;
      }
    }
  }, [messages, isLoadingMore]);

  // Fetch messages when navigating to an existing session.
  useEffect(() => {
    if (!id) {
      setMessages([]);
      setOffset(0);
      setHasMore(false);
      return;
    }

    if (streamingSessionRef.current === id) return;

    let cancelled = false;
    const fetchMessages = async () => {
      const page = await getSessionMessages(id, PAGE_SIZE, 0);
      if (cancelled) return;
      setMessages(page.messages);
      setOffset(PAGE_SIZE);
      setHasMore(page.has_more);
    };

    fetchMessages();
    return () => {
      cancelled = true;
    };
  }, [id]);

  // Scroll-to-top detection for loading older messages.
  useEffect(() => {
    if (!id || !hasMore) return;

    const viewport = scrollRef.current?.querySelector('[data-slot="scroll-area-viewport"]');
    if (!viewport) return;

    const handleScroll = async () => {
      if (viewport.scrollTop !== 0 || isLoadingMore || !hasMore) return;

      setIsLoadingMore(true);
      try {
        const prevHeight = viewport.scrollHeight;
        const page = await getSessionMessages(id, PAGE_SIZE, offset);
        setMessages((prev) => [...page.messages, ...prev]);
        setOffset((o) => o + PAGE_SIZE);
        setHasMore(page.has_more);
        // Restore scroll position after prepend.
        requestAnimationFrame(() => {
          viewport.scrollTop = viewport.scrollHeight - prevHeight;
        });
      } catch (err) {
        console.error("Failed to load older messages", err);
        toast.error("Failed to load older messages");
      } finally {
        setIsLoadingMore(false);
      }
    };

    viewport.addEventListener("scroll", handleScroll);
    return () => viewport.removeEventListener("scroll", handleScroll);
  }, [id, hasMore, isLoadingMore, offset]);

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
        break;
      }

      case "done": {
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

  const handleSend = async (userInput: string, fileToSend: File | null, maxMode: boolean) => {
    if (!userInput) return;

    setIsStreaming(true);

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
        maxMode,
        fileToSend,
        (event: ChatEvent) => {
          handleStreamEvent(event, {
            onSessionId: (sid) => {
              streamingSessionRef.current = sid;

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
      );

      streamingSessionRef.current = null;
      setIsStreaming(false);
    } catch (err) {
      streamingSessionRef.current = null;
      setIsStreaming(false);
      console.error(err);
    }
  };

  const onHandleSuggestion = (suggestion: string) => {
    handleSend(suggestion, null, false);
  };

  return (
    <>
      <SidebarProvider className="h-screen w-full flex-1 overflow-hidden bg-background">
        <SideBar title={title} setTitle={setTitle} sessionVersion={sessionVersion} />
        <main className="flex h-full w-full flex-col overflow-hidden">
          {/* Same chrome as every other route: MainLayout's sticky glass-chrome
              bar. The rail owns the branding, so this row carries only the nav. */}
          <div className="glass-chrome sticky top-0 z-50 w-full shrink-0 border-b border-border">
            <div className="flex min-w-0 items-center gap-4 px-4 py-3">
              <Navbar />
            </div>
          </div>
          {/* Opaque, like WatchList's table and HomePage's cards. A blurred
              translucent panel was the one workspace surface in the app that
              didn't read as a solid sheet of content. */}
          <div className="surface-solid relative m-4 flex h-full flex-1 flex-col overflow-hidden rounded-xl border border-border shadow-sm">
            <Header shouldAnimate={!!id} title={title} setTitle={setTitle} />
            <div ref={scrollRef} className="relative flex h-full w-full flex-1 flex-col overflow-hidden">
              <ScrollArea className="h-full w-full flex-1">
                <div
                  className="mx-auto flex min-h-full w-full max-w-3xl flex-col space-y-6 px-4 pb-36 pt-6 md:pb-40"
                  role="log"
                  aria-live="polite"
                  aria-label="Conversation"
                >
                  {isLoadingMore && (
                    <div className="flex justify-center py-2">
                      <Loader2 className="size-4 animate-spin text-muted-foreground" aria-hidden="true" />
                      <span className="sr-only">Loading older messages</span>
                    </div>
                  )}
                  {messages.length > 0 || id ? (
                    <MessageList messages={messages} />
                  ) : (
                    <div className="mt-8 flex flex-1 flex-col items-center justify-center gap-5 text-center">
                      <img src={stockmindLogo} alt="" width={128} height={128} className="size-32 drop-shadow-sm" />
                      <div className="max-w-md space-y-2">
                        <h2 className="text-xl font-bold tracking-tight text-foreground">Welcome to StockMind</h2>
                        <p className="text-sm leading-relaxed text-muted-foreground">
                          Your AI-powered assistant for market analysis and stock insights. Ask me anything about the
                          stock market.
                        </p>
                      </div>

                      <div className="mt-2 grid w-full max-w-xl grid-cols-1 gap-2 sm:grid-cols-2">
                        {[
                          "What is FPT stock price?",
                          "Should I buy VNM?",
                          "Explain P/E ratio",
                          "get FPT stock price and report",
                        ].map((suggestion, idx) => (
                          <Button
                            key={idx}
                            variant="outline"
                            onClick={() => onHandleSuggestion(suggestion)}
                            className="h-auto min-h-11 justify-start whitespace-normal px-3.5 py-2.5 text-left text-sm font-normal"
                          >
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
            <ChatInput onSend={handleSend} isStreaming={isStreaming} />
          </div>
        </main>
      </SidebarProvider>
      <Toaster position="top-right" />
    </>
  );
};

export default ChatbotPage;
