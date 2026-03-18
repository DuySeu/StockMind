import { getAgentFlows } from "@/api/agent_flows";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { AgentFlow } from "@/types/agent_flow";
import { Box, ExternalLink, GitBranch, Layers, ShieldCheck, ShieldX, Workflow, Terminal } from "lucide-react";
import { useEffect, useState } from "react";

const ConfigValue = ({ value, depth = 0 }: { value: unknown; depth?: number }) => {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground/50 text-xs italic">N/A</span>;
  }

  if (typeof value === "boolean") {
    return (
      <div
        className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ${
          value ? "text-green-600 bg-green-500/10" : "text-red-600 bg-red-500/10"
        }`}
      >
        {value ? <ShieldCheck className="w-3 h-3" /> : <ShieldX className="w-3 h-3" />}
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
    <>
      <div className="flex-1 overflow-hidden flex flex-col w-full min-h-0">
        <div className="flex flex-col gap-3 p-6 backdrop-blur-sm">
          <h1 className="text-3xl md:text-4xl font-bold tracking-tight text-foreground bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text">
            Agent Flows
          </h1>
          <p className="text-muted-foreground text-lg max-w-2xl">Configure and monitor your active agent workflows.</p>
        </div>
        <ScrollArea className="flex-1 w-full min-h-0">
          <div className="flex-1 flex flex-col gap-8 p-4 md:p-6 max-w-7xl mx-auto w-full">
            <div className="flex flex-col gap-6">
              <Accordion type="single" collapsible className="w-full space-y-4">
                {agentFlows.map((agentFlow) => (
                  <AccordionItem
                    value={agentFlow.id}
                    key={agentFlow.id}
                    className="group border border-border/60 bg-card/40 backdrop-blur-sm rounded-xl px-2 overflow-hidden shadow-sm hover:shadow-md hover:border-primary/20 transition-all duration-300"
                  >
                    <AccordionTrigger className="hover:no-underline py-5 px-4 [&[data-state=open]]:text-primary">
                      <div className="flex items-center gap-5 w-full">
                        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/20 group-hover:scale-105 transition-transform duration-300">
                          <GitBranch className="w-6 h-6" />
                        </div>
                        <div className="flex flex-col items-start gap-1">
                          <span className="text-secondary-foreground text-xl font-semibold tracking-tight">
                            {agentFlow.name}
                          </span>
                          <span className="text-xs text-muted-foreground font-mono bg-muted/80 px-2 py-0.5 rounded-md border border-border/50">
                            ID: {agentFlow.id}
                          </span>
                        </div>
                      </div>
                    </AccordionTrigger>

                    <AccordionContent>
                      <div className="border-t border-border/50 p-6">
                        {agentFlow.config && typeof agentFlow.config === "object" ? (
                          // Just a wrapper to ensure layout context
                          <div className="w-full">
                            <ConfigValue value={agentFlow.config} />
                          </div>
                        ) : (
                          <div className="flex items-center justify-center gap-3 text-muted-foreground p-8 bg-muted/10 rounded-xl border border-dashed border-border/60">
                            <Box className="w-5 h-5 opacity-50" />
                            <span className="text-sm font-medium">
                              {agentFlow.config ? (
                                <ConfigValue value={agentFlow.config} />
                              ) : (
                                "No configuration available"
                              )}
                            </span>
                          </div>
                        )}
                      </div>
                    </AccordionContent>
                  </AccordionItem>
                ))}
              </Accordion>

              {agentFlows.length === 0 && (
                <div className="flex flex-col items-center justify-center py-24 text-center gap-6 opacity-60">
                  <div className="p-4 rounded-full bg-muted/50">
                    <Layers className="w-12 h-12 text-muted-foreground" />
                  </div>
                  <div className="space-y-1">
                    <h3 className="text-lg font-medium text-foreground">No flows found</h3>
                    <p className="text-muted-foreground text-sm">
                      There are currently no agent flows available to display.
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </ScrollArea>
      </div>
    </>
  );
};

export default SettingPage;
