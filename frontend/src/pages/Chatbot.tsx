import { sendChatMessage } from "@/api/chat";
import type { ChatEvent } from "@/api/chat";
import { errorMessage, getSessionMessages, isNotFoundError, updateSessionTitle } from "@/api/sessions";
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
import type { Message, Metadata, ToolCallContent } from "@/types/message";
import { isToolCall, statusFromResult, toIsError } from "@/types/message";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

const PAGE_SIZE = 20;
/** Titles derived from a first message get cut here; the server caps at 120. */
const DERIVED_TITLE_MAX = 60;

const deriveTitle = (input: string): string => {
  const flat = input.replace(/\s+/g, " ").trim();
  return flat.length > DERIVED_TITLE_MAX ? `${flat.slice(0, DERIVED_TITLE_MAX - 1)}…` : flat;
};

/**
 * Rewrite the metadata of the last assistant message. Every streamed tool
 * update lands on the turn currently being generated, which is always last.
 */
const patchLastAssistant = (
  messages: Message[],
  patch: (metadata: Metadata[], message: Message) => Metadata[],
): Message[] => {
  const lastIndex = messages.length - 1;
  if (lastIndex < 0 || messages[lastIndex].role !== "assistant") return messages;
  const current = messages[lastIndex];
  const updated = [...messages];
  updated[lastIndex] = { ...current, metadata: patch(current.metadata ?? [], current) };
  return updated;
};

/**
 * Close out tools that never got a `tool_result` — a stream that ended early or
 * was stopped. Without this their chips keep spinning forever.
 */
const settleRunningTools = (metadata: Metadata[], reason: string): Metadata[] =>
  metadata.map((m) =>
    isToolCall(m) && m.status === "running"
      ? { ...m, status: "error" as const, is_error: true, result: m.result ?? reason }
      : m,
  );

/**
 * Record a failure on the turn being generated, creating the assistant turn if
 * the stream died before producing one. Mirrors what the server persists, so the
 * bubble on screen and the bubble after a reload are the same bubble.
 */
const appendTurnError = (messages: Message[], message: string, code?: string): Message[] => {
  const failure: Metadata = { type: "error", message, code };
  const lastIndex = messages.length - 1;
  if (lastIndex < 0 || messages[lastIndex].role !== "assistant") {
    return [...messages, { role: "assistant", content: "", metadata: [failure] }];
  }
  return patchLastAssistant(messages, (metadata) => [...settleRunningTools(metadata, message), failure]);
};

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
  const abortRef = useRef<AbortController | null>(null);

  const refreshSessions = useCallback(() => setSessionVersion((v) => v + 1), []);

  // Abort an in-flight stream if the page unmounts mid-generation.
  useEffect(() => () => abortRef.current?.abort(), []);

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
      try {
        const page = await getSessionMessages(id, PAGE_SIZE, 0);
        if (cancelled) return;
        setMessages(page.messages);
        setOffset(PAGE_SIZE);
        setHasMore(page.has_more);
      } catch (err) {
        if (cancelled) return;
        // A URL pointing at a session that isn't in the DB — deleted elsewhere,
        // or a hand-edited/stale link. Land on a usable new chat rather than an
        // empty shell whose composer would 404 on every send.
        if (isNotFoundError(err)) {
          toast.error("This conversation no longer exists.");
          navigate("/c", { replace: true });
          return;
        }
        console.error("Failed to load conversation", err);
        toast.error(errorMessage(err, "Failed to load this conversation"));
      }
    };

    fetchMessages();
    return () => {
      cancelled = true;
    };
  }, [id, navigate]);

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
        if (!data || typeof data !== "object") break;
        const raw = data as Record<string, unknown>;
        if (typeof raw.id !== "string" || !raw.id) break;

        // Appended, never keyed by name — the metadata array *is* the call
        // order, so three chips render in the sequence the model invoked them.
        const call: ToolCallContent = {
          type: "tool_call",
          id: raw.id,
          name: typeof raw.name === "string" ? raw.name : "unknown",
          arguments: typeof raw.arguments === "string" ? raw.arguments : undefined,
          status: "running",
        };

        setMessages((prev) => {
          const lastIndex = prev.length - 1;
          if (lastIndex < 0 || prev[lastIndex].role !== "assistant") {
            return [...prev, { role: "assistant", content: "", metadata: [call] }];
          }
          return patchLastAssistant(prev, (metadata) =>
            // Re-emitted ids update in place instead of adding a duplicate chip.
            metadata.some((m) => isToolCall(m) && m.id === call.id)
              ? metadata.map((m) => (isToolCall(m) && m.id === call.id ? { ...m, ...call } : m))
              : [...metadata, call],
          );
        });
        break;
      }

      case "tool_result": {
        if (!data || typeof data !== "object") break;
        const raw = data as Record<string, unknown>;
        if (typeof raw.id !== "string" || !raw.id) break;

        // Matched back onto its call by id. The wire carries the output as
        // `result` and the flag as the string `"true"`.
        const isError = toIsError(raw.is_error);
        const result = typeof raw.result === "string" ? raw.result : undefined;

        setMessages((prev) =>
          patchLastAssistant(prev, (metadata) =>
            metadata.map((m) =>
              isToolCall(m) && m.id === raw.id
                ? { ...m, result, is_error: isError, status: statusFromResult(isError) }
                : m,
            ),
          ),
        );
        break;
      }

      case "done": {
        setMessages((prev) =>
          patchLastAssistant(prev, (metadata) =>
            settleRunningTools(metadata, "No result received").map((m) =>
              m.type === "thinking" ? { ...m, is_open: false } : m,
            ),
          ),
        );
        break;
      }

      case "error": {
        const message = typeof data === "string" ? data : ((data as { message?: string })?.message ?? "Stream error");
        const code = typeof data === "object" ? (data as { code?: string })?.code : undefined;
        toast.error(message);
        // Also rendered in the thread: a toast is dismissible and disappears on
        // reload, which left the failed turn looking like it had never replied.
        // `appendTurnError` settles any pending tool on the way.
        setMessages((prev) => appendTurnError(prev, message, code));
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

    const controller = new AbortController();
    abortRef.current = controller;

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
                const derived = deriveTitle(userInput);
                setTitle(derived);
                navigate(`/c/${sid}`, { replace: true });
                // Persist before refreshing the rail. Refreshing first meant the
                // sidebar read the server's placeholder "New conversation" and
                // pushed it straight back over the title just set here — which
                // is why every conversation in the list had the same name.
                updateSessionTitle(sid, derived)
                  .catch((err) => console.error("Failed to save conversation title", err))
                  .finally(refreshSessions);
              }
            },
          });
        },
        (error) => {
          // The server sends `{"error": "..."}` for every rejected request;
          // show that, not a placeholder that hides an unknown session behind
          // the same words as a malformed body.
          const message = errorMessage(error, "An error occurred while processing your request.");
          console.error("Stream error:", error);
          toast.error(message);
          setMessages((prev) => appendTurnError(prev, message));
        },
        controller.signal,
      );
    } catch (err) {
      console.error(err);
      toast.error(errorMessage(err, "An error occurred while processing your request."));
    } finally {
      // A stopped stream leaves tools mid-flight; close them out so no chip is
      // left spinning after the request is gone.
      setMessages((prev) =>
        controller.signal.aborted
          ? patchLastAssistant(prev, (metadata) => settleRunningTools(metadata, "Stopped"))
          : prev,
      );
      abortRef.current = null;
      streamingSessionRef.current = null;
      setIsStreaming(false);
    }
  };

  const handleStop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  // Re-asks the question that failed. The failed turn stays on screen and the
  // retry appends a fresh pair, because that is exactly what the server stored —
  // hiding the attempt here would put the thread back out of step with the DB.
  const handleRetry = (content: string) => {
    if (isStreaming) return;
    void handleSend(content, null, false);
  };

  /** Rename the open conversation, keeping the header and the rail in step. */
  const handleRename = useCallback(
    async (next: string) => {
      if (!id) return;
      const saved = await updateSessionTitle(id, next);
      setTitle(saved);
      refreshSessions();
    },
    [id, refreshSessions],
  );

  const onHandleSuggestion = (suggestion: string) => {
    handleSend(suggestion, null, false);
  };

  return (
    <>
      <SidebarProvider className="h-screen w-full flex-1 overflow-hidden bg-background">
        <SideBar setTitle={setTitle} sessionVersion={sessionVersion} />
        <main className="flex h-full w-full flex-col overflow-hidden">
          <div className="glass-chrome sticky top-0 z-50 w-full shrink-0 border-b border-border">
            <div className="flex min-w-0 items-center gap-4 px-4 py-3">
              <Navbar />
            </div>
          </div>
          <div className="surface-solid relative m-4 flex h-full flex-1 flex-col overflow-hidden rounded-xl border border-border shadow-sm">
            <Header shouldAnimate={!!id} title={title} canRename={!!id} onRename={handleRename} />
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
                    <MessageList messages={messages} onRetry={handleRetry} retryDisabled={isStreaming} />
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
            <ChatInput onSend={handleSend} isStreaming={isStreaming} onStop={handleStop} />
          </div>
        </main>
      </SidebarProvider>
      <Toaster position="top-right" />
    </>
  );
};

export default ChatbotPage;
