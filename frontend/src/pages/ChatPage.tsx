import React from "react";
import { Menu, Send, Upload, StopCircle, Moon, Sun, ChevronDown } from "lucide-react";

type MessageRole = "user" | "assistant";

type Message = {
  id: string;
  role: MessageRole;
  content: string;
  createdAt: string;
};

type Thread = {
  id: string;
  title: string;
  lastActivityAt: string;
};

function classNames(...classes: Array<string | false | null | undefined>) {
  return classes.filter(Boolean).join(" ");
}

const sampleThreads: Thread[] = [
  { id: "t1", title: "Market outlook for Q4", lastActivityAt: new Date().toISOString() },
  { id: "t2", title: "What is P/E ratio?", lastActivityAt: new Date(Date.now() - 3600_000).toISOString() },
  { id: "t3", title: "Build a momentum strategy", lastActivityAt: new Date(Date.now() - 86400_000).toISOString() },
];

const samplePrompts = [
  "What are today's top gainers in tech?",
  "Explain dividend yield like I'm 15",
  "Summarize NVDA latest earnings call",
  "Compare SPY vs. QQQ over 5 years",
];

function formatTime(iso: string) {
  try {
    const d = new Date(iso);
    return d.toLocaleString();
  } catch {
    return iso;
  }
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === "user";
  return (
    <div className={classNames("flex w-full", isUser ? "justify-end" : "justify-start")} aria-live="polite">
      <div
        className={classNames(
          "max-w-[85%] rounded-lg px-4 py-3 shadow-sm text-sm whitespace-pre-wrap",
          isUser ? "bg-primary text-primary-foreground" : "bg-muted text-foreground"
        )}
      >
        {message.content}
        <div
          className={classNames("mt-1 text-[10px] opacity-70", isUser ? "text-primary-foreground" : "text-foreground")}
        >
          {formatTime(message.createdAt)}
        </div>
      </div>
    </div>
  );
}

export default function ChatPage() {
  const [dark, setDark] = React.useState(false);
  const [sidebarOpen, setSidebarOpen] = React.useState(false);
  const [activeThreadId, setActiveThreadId] = React.useState<string | null>(null);
  const [messages, setMessages] = React.useState<Message[]>([]);
  const [input, setInput] = React.useState("");
  const [isStreaming, setIsStreaming] = React.useState(false);
  const [model, setModel] = React.useState("gpt-4o-mini");

  const listRef = React.useRef<HTMLDivElement | null>(null);

  React.useEffect(() => {
    if (!listRef.current) return;
    listRef.current.scrollTop = listRef.current.scrollHeight;
  }, [messages, isStreaming]);

  React.useEffect(() => {
    const root = document.documentElement;
    if (dark) root.classList.add("dark");
    else root.classList.remove("dark");
  }, [dark]);

  function startNewThread(prompt?: string) {
    const id = `t-${Date.now()}`;
    setActiveThreadId(id);
    if (prompt) {
      const now = new Date().toISOString();
      setMessages([{ id: `m-${Date.now()}`, role: "user", content: prompt, createdAt: now }]);
      void handleSend(prompt, id);
    } else {
      setMessages([]);
    }
  }

  async function handleSend(text?: string, threadId?: string) {
    const content = (text ?? input).trim();
    if (!content) return;
    const now = new Date().toISOString();
    const newMsg: Message = { id: `m-${Date.now()}`, role: "user", content, createdAt: now };
    setMessages((prev) => [...prev, newMsg]);
    setInput("");
    setIsStreaming(true);

    // Simulate assistant streaming. Replace with real API call.
    await new Promise((r) => setTimeout(r, 500));
    const reply: Message = {
      id: `m-${Date.now() + 1}`,
      role: "assistant",
      content: `You said: "${content}"\n\nThis is a placeholder response.`,
      createdAt: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, reply]);
    setIsStreaming(false);
  }

  function handleStop() {
    setIsStreaming(false);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSend();
    }
  }

  const activeThread: Thread | undefined = activeThreadId
    ? sampleThreads.find((t) => t.id === activeThreadId) ?? {
        id: activeThreadId,
        title: "New chat",
        lastActivityAt: new Date().toISOString(),
      }
    : undefined;

  return (
    <div className="flex h-svh w-svw overflow-hidden bg-background text-foreground">
      {/* Mobile Top Bar */}
      <div className="lg:hidden fixed inset-x-0 top-0 z-30 flex h-14 items-center justify-between border-b bg-background px-3">
        <button
          type="button"
          aria-label="Toggle sidebar"
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border hover:bg-accent"
          onClick={() => setSidebarOpen((v) => !v)}
        >
          <Menu className="h-5 w-5" />
        </button>
        <div className="text-sm font-medium">Chatbot</div>
        <button
          type="button"
          aria-label="Toggle theme"
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border hover:bg-accent"
          onClick={() => setDark((v) => !v)}
        >
          {dark ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
        </button>
      </div>

      {/* Sidebar */}
      <aside
        className={classNames(
          "fixed z-20 flex h-svh w-72 flex-col border-r bg-sidebar text-sidebar-foreground transition-transform lg:static",
          "lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0",
          "pt-14 lg:pt-0"
        )}
        aria-label="Thread history"
      >
        <div className="flex h-14 items-center justify-between border-b px-4 lg:hidden">
          <div className="text-sm font-medium">Threads</div>
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1 rounded-md border px-2 text-xs hover:bg-accent"
            onClick={() => startNewThread()}
          >
            New chat
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-3">
          <button
            type="button"
            className="mb-3 inline-flex h-9 w-full items-center justify-center rounded-md border text-sm hover:bg-accent"
            onClick={() => startNewThread()}
          >
            New chat
          </button>
          <div className="space-y-1">
            {sampleThreads.map((t) => (
              <button
                key={t.id}
                type="button"
                className={classNames(
                  "w-full rounded-md border px-3 py-2 text-left text-sm hover:bg-accent",
                  activeThreadId === t.id && "ring-1 ring-sidebar-ring"
                )}
                onClick={() => setActiveThreadId(t.id)}
              >
                <div className="truncate font-medium">{t.title}</div>
                <div className="mt-0.5 text-xs text-muted-foreground">{formatTime(t.lastActivityAt)}</div>
              </button>
            ))}
          </div>
        </div>
        <div className="border-t p-3 hidden lg:block">
          <button
            type="button"
            className="inline-flex h-9 w-full items-center justify-center gap-2 rounded-md border text-sm hover:bg-accent"
            onClick={() => setDark((v) => !v)}
          >
            {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            Toggle theme
          </button>
        </div>
      </aside>

      {/* Main */}
      <main className="flex min-w-0 flex-1 flex-col">
        <header className="hidden h-14 shrink-0 items-center justify-between border-b px-4 lg:flex">
          <div className="flex items-center gap-2">
            <div className="text-sm font-semibold">Chatbot</div>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <button
                type="button"
                aria-haspopup="listbox"
                aria-expanded="false"
                className="inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm hover:bg-accent"
              >
                {model}
                <ChevronDown className="h-4 w-4" />
              </button>
            </div>
            <button
              type="button"
              aria-label="Toggle theme"
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border hover:bg-accent"
              onClick={() => setDark((v) => !v)}
            >
              {dark ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
            </button>
          </div>
        </header>

        {/* Chat Area */}
        <div className="flex min-h-0 flex-1 flex-col">
          {!activeThread && messages.length === 0 ? (
            <div className="flex flex-1 items-center justify-center p-6">
              <div className="mx-auto max-w-2xl text-center">
                <div className="animate-in fade-in zoom-in-95 slide-in-from-bottom-2 duration-500">
                  <h1 className="text-xl font-semibold tracking-tight">
                    Welcome back! What can I help you with today?
                  </h1>
                  <p className="mt-2 text-sm text-muted-foreground">
                    Start with an example below or type your own question.
                  </p>
                </div>
                <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  {samplePrompts.map((p) => (
                    <button
                      key={p}
                      type="button"
                      className="rounded-lg border p-3 text-left text-sm hover:bg-accent"
                      onClick={() => startNewThread(p)}
                    >
                      {p}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <>
              <div ref={listRef} className="flex-1 space-y-3 overflow-y-auto p-4">
                {messages.map((m) => (
                  <MessageBubble key={m.id} message={m} />
                ))}
                {isStreaming && (
                  <div className="flex w-full justify-start">
                    <div className="inline-flex items-center gap-2 rounded-lg bg-muted px-3 py-2 text-xs">
                      <div className="h-2 w-2 animate-pulse rounded-full bg-foreground" />
                      Generating...
                    </div>
                  </div>
                )}
              </div>
              <div className="border-t p-3">
                <div className="mx-auto flex max-w-3xl items-end gap-2">
                  <button
                    type="button"
                    aria-label="Upload"
                    className="inline-flex h-10 w-10 items-center justify-center rounded-md border hover:bg-accent"
                  >
                    <Upload className="h-5 w-5" />
                  </button>
                  <textarea
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={onKeyDown}
                    rows={1}
                    placeholder="Send a message..."
                    className="min-h-10 max-h-40 flex-1 resize-y rounded-md border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  />
                  {isStreaming ? (
                    <button
                      type="button"
                      aria-label="Stop generating"
                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border hover:bg-accent"
                      onClick={handleStop}
                    >
                      <StopCircle className="h-5 w-5" />
                    </button>
                  ) : (
                    <button
                      type="button"
                      aria-label="Send"
                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border hover:bg-accent"
                      onClick={() => void handleSend()}
                    >
                      <Send className="h-5 w-5" />
                    </button>
                  )}
                </div>
                <div className="mx-auto mt-2 max-w-3xl text-xs text-muted-foreground">
                  Press Enter to send, Shift + Enter for new line.
                </div>
              </div>
            </>
          )}
        </div>
      </main>
    </div>
  );
}
