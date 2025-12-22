import { getAgentFlows } from "@/api/agent_flows";
import Header from "@/components/containers/Header";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatContext } from "@/hooks/context";
import { Bolt } from "lucide-react";
import { useEffect, useState } from "react";

const ConfigRenderer = ({ value }: { value: any }) => {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground/50 text-xs italic">N/A</span>;
  }

  if (typeof value === "boolean") {
    return (
      <span
        className={`text-xs font-medium px-1.5 py-0.5 rounded ${
          value ? "bg-green-500/10 text-green-600" : "bg-red-500/10 text-red-600"
        }`}
      >
        {value ? "TRUE" : "FALSE"}
      </span>
    );
  }

  if (Array.isArray(value)) {
    if (value.length === 0) return <span className="text-muted-foreground/50 text-xs italic">Empty List</span>;
    return (
      <div className="flex flex-col gap-2 mt-1.5">
        {value.map((item, idx) => (
          <div key={idx} className="pl-3 border-l-2 border-border/40">
            <ConfigRenderer value={item} />
          </div>
        ))}
      </div>
    );
  }

  if (typeof value === "object") {
    if (Object.keys(value).length === 0)
      return <span className="text-muted-foreground/50 text-xs italic">Empty Object</span>;
    return (
      <div className="flex flex-col gap-2 mt-1.5 w-full">
        {Object.entries(value).map(([k, v]) => (
          <div key={k} className="flex flex-col gap-1 w-full bg-muted/20 p-2 rounded-md border border-border/30">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {k.replace(/_/g, " ")}:
            </span>
            <div className="pl-1 text-sm text-foreground/90 w-full overflow-hidden">
              <ConfigRenderer value={v} />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (typeof value === "string" && (value.startsWith("http") || value.startsWith("https"))) {
    return (
      <a href={value} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline break-all">
        {value}
      </a>
    );
  }

  return <span className="break-words whitespace-pre-wrap">{String(value)}</span>;
};

const SettingPage = () => {
  const [agentFlows, setAgentFlows] = useState<any[]>([]);
  const { setTitle } = useChatContext();

  useEffect(() => {
    setTitle("Settings");
  }, []);

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
      <Header icon={<Bolt className="text-primary w-6 h-6" />} />

      <div className="flex-1 overflow-hidden flex flex-col w-full min-h-0">
        <ScrollArea className="flex-1 w-full min-h-0">
          <div className="flex-1 flex flex-col gap-6 p-6">
            <h1 className="text-3xl font-bold tracking-tight text-secondary-foreground">Agent Flows</h1>

            <Accordion type="single" collapsible className="w-full space-y-4">
              {agentFlows.map((agentFlow) => (
                <AccordionItem
                  value={agentFlow.id}
                  key={agentFlow.id}
                  className="bg-card border border-border rounded-xl px-4 overflow-hidden shadow-sm"
                >
                  <AccordionTrigger className="hover:no-underline py-4">
                    <span className="text-lg font-medium text-secondary-foreground">{agentFlow.name}</span>
                  </AccordionTrigger>
                  <AccordionContent className="pb-4">
                    {agentFlow.config && typeof agentFlow.config === "object" ? (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pt-2">
                        {Object.entries(agentFlow.config).map(([key, value]) => (
                          <div
                            key={key}
                            className="bg-muted/30 p-3 rounded-lg border border-border/50 flex flex-col overflow-hidden"
                          >
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                              {key.replace(/_/g, " ")}
                            </h4>
                            <div className="text-sm text-foreground font-medium w-full">
                              <ConfigRenderer value={value} />
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="bg-muted/30 p-4 rounded-lg text-sm font-mono text-muted-foreground">
                        <ConfigRenderer value={agentFlow.config} />
                      </div>
                    )}
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </div>
        </ScrollArea>
      </div>
    </>
  );
};

export default SettingPage;
