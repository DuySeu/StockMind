import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import type { EvalCatalog, EvalModule } from "@/api/eval";

export type ModuleId = "chat" | EvalModule;

export type EvalRequest =
  | { module: "chat"; content: string; sessionId: string | undefined; maxMode: boolean }
  | { module: EvalModule; input: Record<string, unknown> };

interface Props {
  module: ModuleId;
  catalog: EvalCatalog | undefined;
  sessionId: string | undefined;
  isRunning: boolean;
  onRun: (request: EvalRequest) => void;
}

// Every field of every module, kept together so switching modules keeps what was typed
const EMPTY_FIELDS = {
  question: "",
  maxMode: false,
  continueSession: false,
  message: "",
  history: "",
  source: "get_news",
  content: "",
  goal: "",
  agent: "",
  task: "",
  agentGoal: "",
  context: "",
  tool: "",
  args: "{}",
};

const LABEL = "flex flex-col gap-1 text-xs font-medium text-muted-foreground";

// Render the input form of one module and turn it into an eval request
export function ModuleForm({ module, catalog, sessionId, isRunning, onRun }: Props) {
  const [fields, setFields] = useState(EMPTY_FIELDS);
  const [formError, setFormError] = useState<string>();

  // Default the pickers to the first catalog entry
  const agentName = fields.agent || catalog?.agents[0]?.name || "";
  const toolName = fields.tool || catalog?.tools[0]?.name || "";
  const agentInfo = catalog?.agents.find((a) => a.name === agentName);
  const toolInfo = catalog?.tools.find((t) => t.name === toolName);

  const set = (key: keyof typeof EMPTY_FIELDS) => (value: string | boolean) =>
    setFields((prev) => ({ ...prev, [key]: value }));

  // Build the request, rejecting input the server would refuse
  const buildRequest = (): EvalRequest | string => {
    switch (module) {
      case "chat":
        if (!fields.question.trim()) return "Question is required";
        return {
          module,
          content: fields.question,
          sessionId: fields.continueSession ? sessionId : undefined,
          maxMode: fields.maxMode,
        };
      case "guardrail": {
        if (!fields.message.trim()) return "Message is required";
        const history = fields.history
          .split("\n")
          .map((line) => line.trim())
          .filter(Boolean)
          .map((line) => {
            const [role, ...rest] = line.split(":");
            return { role: role.trim(), content: rest.join(":").trim() };
          });
        if (history.some((m) => m.role !== "user" && m.role !== "assistant")) {
          return "History lines must start with user: or assistant:";
        }
        return { module, input: { message: fields.message, history } };
      }
      case "sanitize":
        if (!fields.source.trim() || !fields.content.trim()) return "Source and content are required";
        return { module, input: { source: fields.source, content: fields.content } };
      case "plan":
      case "pipeline":
        if (!fields.goal.trim()) return "Goal is required";
        return { module, input: { goal: fields.goal } };
      case "agent": {
        if (!agentName || !fields.task.trim()) return "Agent and task are required";
        let context: unknown = undefined;
        if (fields.context.trim()) {
          try {
            context = JSON.parse(fields.context);
          } catch {
            return "Context must be a JSON object of step id to output";
          }
        }
        return { module, input: { agent: agentName, task: fields.task, goal: fields.agentGoal, context } };
      }
      case "tool": {
        if (!toolName) return "Tool is required";
        try {
          return { module, input: { tool: toolName, args: JSON.parse(fields.args || "{}") } };
        } catch {
          return "Args must be valid JSON";
        }
      }
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const request = buildRequest();
    if (typeof request === "string") {
      setFormError(request);
      return;
    }
    setFormError(undefined);
    onRun(request);
  };

  return (
    <form className="flex flex-wrap items-end gap-3 border-b p-4" onSubmit={handleSubmit}>
      {module === "chat" && (
        <>
          <label className={cn(LABEL, "min-w-[280px] flex-1")}>
            Question
            <Textarea value={fields.question} onChange={(e) => set("question")(e.target.value)} rows={2} className="text-sm" />
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={fields.maxMode} onChange={(e) => set("maxMode")(e.target.checked)} />
            max_mode
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={fields.continueSession}
              disabled={!sessionId}
              onChange={(e) => set("continueSession")(e.target.checked)}
            />
            continue session {sessionId ? <span className="font-mono text-xs">{sessionId.slice(0, 8)}</span> : null}
          </label>
        </>
      )}

      {module === "guardrail" && (
        <>
          <label className={cn(LABEL, "min-w-[260px] flex-1")}>
            Message
            <Textarea value={fields.message} onChange={(e) => set("message")(e.target.value)} rows={3} className="text-sm" />
          </label>
          <label className={cn(LABEL, "min-w-[260px] flex-1")}>
            History (optional, one turn per line: user: ... / assistant: ...)
            <Textarea value={fields.history} onChange={(e) => set("history")(e.target.value)} rows={3} className="font-mono text-xs" />
          </label>
          {catalog && !catalog.guardrailEnabled && (
            <p className="w-full text-xs text-destructive">Classifier is off on the server (GUARDRAIL_ENABLED=false): only redact runs.</p>
          )}
        </>
      )}

      {module === "sanitize" && (
        <>
          <label className={cn(LABEL, "w-48")}>
            Source (tool name)
            <Input value={fields.source} onChange={(e) => set("source")(e.target.value)} className="text-sm" />
          </label>
          <label className={cn(LABEL, "min-w-[280px] flex-1")}>
            Tool output
            <Textarea value={fields.content} onChange={(e) => set("content")(e.target.value)} rows={4} className="font-mono text-xs" />
          </label>
        </>
      )}

      {(module === "plan" || module === "pipeline") && (
        <label className={cn(LABEL, "min-w-[280px] flex-1")}>
          Goal
          <Textarea value={fields.goal} onChange={(e) => set("goal")(e.target.value)} rows={2} className="text-sm" />
        </label>
      )}

      {module === "agent" && (
        <>
          <label className={cn(LABEL, "w-48")}>
            Agent
            <select
              value={agentName}
              onChange={(e) => set("agent")(e.target.value)}
              className="h-9 rounded-md border bg-background px-2 text-sm"
            >
              {catalog?.agents.map((a) => (
                <option key={a.name} value={a.name}>
                  {a.name}
                </option>
              ))}
            </select>
          </label>
          <label className={cn(LABEL, "min-w-[240px] flex-1")}>
            Task
            <Textarea value={fields.task} onChange={(e) => set("task")(e.target.value)} rows={2} className="text-sm" />
          </label>
          <label className={cn(LABEL, "min-w-[200px] flex-1")}>
            Goal (optional, defaults to task)
            <Textarea value={fields.agentGoal} onChange={(e) => set("agentGoal")(e.target.value)} rows={2} className="text-sm" />
          </label>
          <label className={cn(LABEL, "min-w-[200px] flex-1")}>
            Context (optional JSON: step id to output)
            <Textarea value={fields.context} onChange={(e) => set("context")(e.target.value)} rows={2} className="font-mono text-xs" />
          </label>
          {agentInfo && (
            <p className="w-full text-xs text-muted-foreground">
              {agentInfo.description} Tools: {agentInfo.tools?.join(", ") || "none"}
            </p>
          )}
        </>
      )}

      {module === "tool" && (
        <>
          <label className={cn(LABEL, "w-56")}>
            Tool
            <select
              value={toolName}
              onChange={(e) => set("tool")(e.target.value)}
              className="h-9 rounded-md border bg-background px-2 text-sm"
            >
              {catalog?.tools.map((t) => (
                <option key={t.name} value={t.name}>
                  {t.name}
                </option>
              ))}
            </select>
          </label>
          <label className={cn(LABEL, "min-w-[280px] flex-1")}>
            Args (JSON)
            <Textarea value={fields.args} onChange={(e) => set("args")(e.target.value)} rows={3} className="font-mono text-xs" />
          </label>
          {toolInfo && (
            <details className="w-full text-xs text-muted-foreground">
              <summary className="cursor-pointer">{toolInfo.description}</summary>
              <pre className="mt-1 max-h-48 overflow-auto rounded-md bg-muted p-2 font-mono">
                {JSON.stringify(toolInfo.schema, null, 2)}
              </pre>
            </details>
          )}
        </>
      )}

      <Button type="submit" disabled={isRunning}>
        {isRunning ? "Running..." : "Run"}
      </Button>
      {formError && <p className="w-full text-xs text-destructive">{formError}</p>}
    </form>
  );
}
