import { getAgentFlows } from "@/api/agent_flows";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import type { AgentFlow } from "@/types/agent_flow";
import { Box, ExternalLink, GitBranch, Layers, ShieldCheck, ShieldX, Workflow, Terminal } from "lucide-react";
import { useEffect, useState } from "react";

// Render one agent-flow config value, recursing into objects and arrays
const ConfigValue = ({ value, depth = 0 }: { value: unknown; depth?: number }) => {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground/50 text-xs italic">N/A</span>;
  }

  if (typeof value === "boolean") {
    return (
      <div
        className={`inline-flex items-center gap-1.5 rounded px-2 py-0.5 text-xs font-medium ${
          value ? "bg-status-ok-bg text-status-ok" : "bg-status-error-bg text-status-error"
        }`}
      >
        {value ? (
          <ShieldCheck className="size-3" aria-hidden="true" />
        ) : (
          <ShieldX className="size-3" aria-hidden="true" />
        )}
        {value ? "True" : "False"}
      </div>
    );
  }

  if (Array.isArray(value)) {
    if (value.length === 0) return <span className="text-muted-foreground/50 text-xs italic">Empty List</span>;
    return (
      <div className="flex flex-col bg-background/50 p-4 rounded-xl border border-border/40 gap-2">
        {value.map((item, idx) => (
          <div key={idx} className="relative pl-4 border-l border-border/60 hover:border-primary/40 transition-colors">
            <div className="absolute -left-[1px] top-2.5 w-1.5 h-1.5 rounded-full bg-border" />
            <ConfigValue value={item} depth={depth + 1} />
          </div>
        ))}
      </div>
    );
  }

  if (typeof value === "object") {
    const obj = value as Record<string, unknown>;
    const keys = Object.keys(obj);
    if (keys.length === 0) return <span className="text-muted-foreground/50 text-xs italic">Empty Object</span>;

    // Depth 0: The Root Grid Container
    // We want a clear separation (e.g. Agents vs Nodes)
    if (depth === 0) {
      return (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 w-full">
          {keys.map((k) => (
            <div key={k} className="flex flex-col gap-3 min-w-0">
              <div className="flex items-center gap-2 pb-2 border-b border-border/40">
                <Workflow className="w-4 h-4 text-primary opacity-80" />
                <h3 className="text-sm font-bold tracking-wider text-foreground uppercase">{k.replace(/_/g, " ")}</h3>
              </div>
              <ConfigValue value={obj[k]} depth={depth + 1} />
            </div>
          ))}
        </div>
      );
    }

    // Depth 1: Inside the main columns (e.g., individual Agent or Node definitions)
    // Use card-like distinct blocks to separate items if it's a map/object
    if (depth === 1) {
      return (
        <div className="flex flex-col gap-4">
          {keys.map((k) => (
            <div
              key={k}
              className="flex flex-col gap-2 p-3 rounded-lg bg-background/50 border border-border/40 shadow-sm"
            >
              <div className="flex items-center gap-2 mb-1">
                <Terminal className="w-3 h-3 text-muted-foreground" />
                <span className="text-xs font-bold text-muted-foreground uppercase">{k.replace(/_/g, " ")}</span>
              </div>
              <div className="pl-1">
                <ConfigValue value={obj[k]} depth={depth + 1} />
              </div>
            </div>
          ))}
        </div>
      );
    }

    // Deep Levels: Clean list
    return (
      <div className="flex flex-col gap-2 mt-1 w-full">
        {keys.map((k) => (
          <div key={k} className="flex flex-col gap-1 w-full">
            <div className="flex items-baseline gap-2">
              <span className="text-xs font-medium text-muted-foreground/70 shrink-0">{k.replace(/_/g, " ")}:</span>
              {/* If simple value, render inline. If complex, render next line */}
              {typeof obj[k] !== "object" && (
                <div className="text-sm text-foreground/90 font-mono">
                  <ConfigValue value={obj[k]} depth={depth + 1} />
                </div>
              )}
            </div>
            {/* Complex values get their own block */}
            {typeof obj[k] === "object" && obj[k] !== null && (
              <div className="pl-3 border-l-2 border-muted-foreground/10 ml-1 my-1">
                <ConfigValue value={obj[k]} depth={depth + 1} />
              </div>
            )}
          </div>
        ))}
      </div>
    );
  }

  if (typeof value === "string" && (value.startsWith("http") || value.startsWith("https"))) {
    return (
      <a
        href={value}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1.5 text-primary hover:text-primary/80 hover:underline transition-colors break-all group"
      >
        <ExternalLink className="w-3 h-3" />
        {value}
      </a>
    );
  }

  return <span className="text-sm text-foreground/90 break-words whitespace-pre-wrap">{String(value)}</span>;
};

// Render every registered agent flow as an expandable config panel
const SettingPage = () => {
  const [agentFlows, setAgentFlows] = useState<AgentFlow[]>([]);

  const fetchAgentFlows = async () => {
    try {
      const response = await getAgentFlows();
      setAgentFlows(response);
    } catch (error) {
      console.log(error);
    }
  };

  useEffect(() => {
    fetchAgentFlows();
  }, []);

  return (
    <div className="flex w-full flex-1 flex-col">
      <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl flex-col gap-1 px-4 py-5 sm:px-6 lg:px-8">
          <h1 className="text-xl font-bold tracking-tight">Agent Flows</h1>
          <p className="text-sm text-muted-foreground">The agent graphs this workspace can run, and their config.</p>
        </div>
      </header>

      {/* The old shell nested a max-w-6xl bg-secondary box around a ScrollArea
          around a max-w-7xl box — two competing widths and a scroll container
          that could never scroll, since nothing constrained its height. */}
      <div className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {agentFlows.length > 0 ? (
          <Accordion type="single" collapsible className="w-full space-y-4">
            {agentFlows.map((agentFlow) => (
              <AccordionItem
                value={agentFlow.id}
                key={agentFlow.id}
                className="glass group overflow-hidden rounded-xl px-2 transition-colors hover:border-ring"
              >
                <AccordionTrigger className="px-4 py-5 hover:no-underline [&[data-state=open]]:text-primary">
                  <div className="flex w-full items-center gap-4">
                    <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
                      <GitBranch className="size-5" aria-hidden="true" />
                    </span>
                    <span className="flex min-w-0 flex-col items-start gap-1">
                      <span className="text-lg font-semibold tracking-tight">{agentFlow.name}</span>
                      <span className="truncate rounded-md border border-border bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
                        {agentFlow.id}
                      </span>
                    </span>
                  </div>
                </AccordionTrigger>

                <AccordionContent>
                  <div className="border-t border-border p-6">
                    {agentFlow.config ? (
                      <ConfigValue value={agentFlow.config} />
                    ) : (
                      <div className="flex items-center justify-center gap-3 rounded-lg border border-dashed border-border p-8 text-sm text-muted-foreground">
                        <Box className="size-5 opacity-50" aria-hidden="true" />
                        This flow has no configuration stored.
                      </div>
                    )}
                  </div>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        ) : (
          /* A composed empty state rather than a dimmed icon: it says what the
             page will hold and where flows come from. */
          <div className="flex flex-col items-center gap-4 rounded-xl border border-dashed border-border px-6 py-20 text-center">
            <span className="flex size-12 items-center justify-center rounded-lg bg-secondary">
              <Layers className="size-6 text-muted-foreground" aria-hidden="true" />
            </span>
            <div className="space-y-1">
              <h2 className="font-semibold">No agent flows yet</h2>
              <p className="mx-auto max-w-sm text-sm leading-relaxed text-muted-foreground">
                Flows are seeded from the backend on startup. Once one is registered it appears here with its full
                agent and node configuration.
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default SettingPage;
