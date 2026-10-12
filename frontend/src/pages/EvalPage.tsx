import { useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { JsonPayload } from "@/components/eval/JsonPayload";
import { ModuleForm, type EvalRequest, type ModuleId } from "@/components/eval/ModuleForm";
import { SpanDetail } from "@/components/eval/SpanDetail";
import { SpanWaterfall } from "@/components/eval/SpanWaterfall";
import { sendChatMessage, type ChatEvent } from "@/api/chat";
import { getEvalCatalog, runEvalModule, type EvalCatalog } from "@/api/eval";
import { buildSpanTree, type SpanNode, type TraceEvent } from "@/lib/trace";

type RunStatus =
  | { state: "idle" }
  | { state: "running" }
  | { state: "done" }
  | { state: "error"; message: string };

// What each module runs, in the order the chat path reaches them
const MODULES: { id: ModuleId; label: string; hint: string }[] = [
  { id: "chat", label: "Full chat", hint: "POST /v1/chat with trace: the whole turn, saved like a real chat" },
  { id: "guardrail", label: "Guardrail", hint: "Redact + topic classify, nothing after" },
  { id: "sanitize", label: "Sanitize", hint: "Clean one tool output and show what the model receives" },
  { id: "plan", label: "Planner", hint: "Plan only, no agent runs" },
  { id: "pipeline", label: "Pipeline", hint: "Plan and run every agent, no guardrail, no DB" },
  { id: "agent", label: "Agent", hint: "One agent on one task, with its LLM rounds and tools" },
  { id: "tool", label: "Tool / MCP", hint: "One tool call with raw args, native or MCP-bridged" },
];

// Run one module, or the full chat turn, and show its spans and result
const EvalPage = () => {
  const [module, setModule] = useState<ModuleId>("chat");
  const [catalog, setCatalog] = useState<EvalCatalog>();
  const [catalogError, setCatalogError] = useState<string>();
  const [sessionId, setSessionId] = useState<string>();
  const [events, setEvents] = useState<TraceEvent[]>([]);
  const [answer, setAnswer] = useState("");
  const [result, setResult] = useState<unknown>();
  const [status, setStatus] = useState<RunStatus>({ state: "idle" });
  const [selectedId, setSelectedId] = useState<string>();
  const abortRef = useRef<AbortController | null>(null);

  // Derive the tree and the summary from the raw events
  const roots = buildSpanTree(events);
  const byId = new Map<string, SpanNode>();
  const index = (node: SpanNode) => {
    byId.set(node.span.id, node);
    node.children.forEach(index);
  };
  roots.forEach(index);
  const root = roots[0];
  const classify = [...byId.values()].find((node) => node.span.name === "guardrail.classify");
  const verdict = classify?.span.output as { Allowed?: boolean; Category?: string } | undefined;
  const summary = events.find((event) => event.phase === "summary");
  const dropped = summary?.phase === "summary" ? (summary.dropped ?? 0) : 0;
  const hint = MODULES.find((m) => m.id === module)?.hint;

  const handleRun = async (request: EvalRequest) => {
    // Reset the previous run
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setEvents([]);
    setAnswer("");
    setResult(undefined);
    setSelectedId(undefined);
    setStatus({ state: "running" });

    let failure: string | undefined;
    const handleEvent = (event: ChatEvent) => {
      switch (event.type) {
        case "start":
          setSessionId((event.data as { session_id?: string }).session_id);
          break;
        case "text":
          setAnswer((prev) => prev + String(event.data ?? ""));
          break;
        case "trace":
          setEvents((prev) => [...prev, event.data as TraceEvent]);
          break;
        case "result":
          setResult(event.data);
          break;
        case "error":
          failure = (event.data as { message?: string })?.message ?? "The run failed";
          break;
      }
    };
    const handleError = (error: unknown) => {
      failure = error instanceof Error ? error.message : String(error);
    };

    // The full chat goes through the real chat API; a module goes through its eval endpoint
    if (request.module === "chat") {
      await sendChatMessage(request.content, request.sessionId, request.maxMode, null, handleEvent, handleError, controller.signal, {
        trace: true,
      });
    } else {
      await runEvalModule(request.module, request.input, handleEvent, handleError, controller.signal);
    }
    if (controller.signal.aborted) return;
    setStatus(failure ? { state: "error", message: failure } : { state: "done" });
  };

  useEffect(() => {
    getEvalCatalog()
      .then(setCatalog)
      .catch((error: unknown) => setCatalogError(error instanceof Error ? error.message : String(error)));
  }, []);

  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      <nav aria-label="Eval module" className="flex flex-wrap items-center gap-1 border-b px-4 pt-3 pb-2">
        {MODULES.map((m) => (
          <Button
            key={m.id}
            type="button"
            size="sm"
            variant={module === m.id ? "secondary" : "ghost"}
            aria-pressed={module === m.id}
            onClick={() => setModule(m.id)}
          >
            {m.label}
          </Button>
        ))}
        <span className="ml-2 text-xs text-muted-foreground">{hint}</span>
        {catalogError && <span className="ml-auto text-xs text-destructive">catalog: {catalogError}</span>}
      </nav>

      <ModuleForm
        module={module}
        catalog={catalog}
        sessionId={sessionId}
        isRunning={status.state === "running"}
        onRun={handleRun}
      />

      <div className="flex flex-wrap gap-4 border-b px-4 py-2 text-xs text-muted-foreground">
        <span>Total {root && !root.isRunning ? `${(root.span.duration_ms / 1000).toFixed(2)}s` : "..."}</span>
        <span>{byId.size} spans</span>
        {verdict && (
          <span>
            guardrail: {verdict.Allowed ? "allowed" : "blocked"} ({verdict.Category || "fail-open"})
          </span>
        )}
        <span>{dropped} dropped</span>
        {status.state === "error" && <span className="text-destructive">{status.message}</span>}
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[3fr_2fr]">
        <div className="min-h-0 overflow-auto border-r">
          {/* Span ids restart at 1 every run, so a new run remounts to drop old collapse state */}
          <SpanWaterfall key={root?.span.start} roots={roots} selectedId={selectedId} onSelect={setSelectedId} />
        </div>
        <div className="min-h-0 overflow-auto">
          <SpanDetail node={selectedId ? byId.get(selectedId) : undefined} />
        </div>
      </div>

      <section className="max-h-[30vh] overflow-auto border-t p-4">
        {answer ? (
          <>
            <h2 className="mb-2 text-xs font-medium text-muted-foreground">Final answer</h2>
            <div className="text-sm leading-relaxed">
              <Markdown remarkPlugins={[remarkGfm]}>{answer}</Markdown>
            </div>
          </>
        ) : result !== undefined ? (
          <JsonPayload label="Result" value={result} />
        ) : (
          <p className="text-sm text-muted-foreground">No result yet.</p>
        )}
      </section>
    </div>
  );
};

export default EvalPage;
