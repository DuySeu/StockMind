import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import type { SpanNode } from "@/lib/trace";

interface Props {
  roots: readonly SpanNode[];
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}

// Bar colour per span kind; unknown kinds fall back to the handler colour
const KIND_COLORS: Record<string, string> = {
  handler: "bg-muted-foreground",
  guardrail: "bg-price-ceiling",
  orchestrator: "bg-chart-2",
  agent: "bg-chart-3",
  llm: "bg-chart-1",
  tool: "bg-chart-4",
  mcp: "bg-chart-5",
};

const INDENT_PX = 14;
// Room left of each name for the collapse toggle
const TOGGLE_PX = 16;

// Render the span tree as indented rows with bars on one shared time axis
export function SpanWaterfall({ roots, selectedId, onSelect }: Props) {
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(new Set());

  // Flatten the visible part of the tree, depth first
  const rows: { node: SpanNode; depth: number }[] = [];
  const visit = (node: SpanNode, depth: number) => {
    rows.push({ node, depth });
    if (!collapsed.has(node.span.id)) node.children.forEach((child) => visit(child, depth + 1));
  };
  roots.forEach((root) => visit(root, 0));

  // The axis runs from the first start to the last known end
  const all: SpanNode[] = [];
  const collect = (node: SpanNode) => {
    all.push(node);
    node.children.forEach(collect);
  };
  roots.forEach(collect);
  const t0 = Math.min(...all.map((n) => n.startMs));
  const tEnd = Math.max(...all.map((n) => n.startMs + n.span.duration_ms));
  const range = Math.max(tEnd - t0, 1);

  const handleToggle = (id: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  if (all.length === 0) {
    return <p className="p-4 text-sm text-muted-foreground">No spans yet.</p>;
  }

  return (
    <div className="text-xs">
      <div className="grid grid-cols-[minmax(220px,2fr)_64px_3fr] gap-2 border-b px-2 py-1 font-medium text-muted-foreground">
        <span>Span</span>
        <span className="text-right">ms</span>
        <span className="flex justify-between">
          <span>0s</span>
          <span>{(range / 1000).toFixed(1)}s</span>
        </span>
      </div>
      <ul>
        {rows.map(({ node, depth }) => {
          const { span } = node;
          const hasChildren = node.children.length > 0;
          const left = ((node.startMs - t0) / range) * 100;
          const width = node.isRunning ? 100 - left : Math.max((span.duration_ms / range) * 100, 0.5);
          return (
            <li key={span.id} className="relative">
              <button
                type="button"
                onClick={() => onSelect(span.id)}
                className={cn(
                  "grid w-full grid-cols-[minmax(220px,2fr)_64px_3fr] items-center gap-2 px-2 py-1 text-left hover:bg-muted",
                  selectedId === span.id && "bg-secondary",
                )}
              >
                <span
                  className={cn("truncate font-mono", span.error && "text-destructive")}
                  style={{ paddingLeft: depth * INDENT_PX + TOGGLE_PX }}
                >
                  {span.name}
                </span>
                <span className="text-right font-mono tabular-nums">{node.isRunning ? "..." : span.duration_ms}</span>
                <span className="relative h-3">
                  <span
                    className={cn(
                      "absolute top-0 h-3 rounded-sm",
                      span.error ? "bg-destructive" : (KIND_COLORS[span.kind] ?? KIND_COLORS.handler),
                      node.isRunning && "animate-pulse opacity-60",
                    )}
                    style={{ left: `${left}%`, width: `${width}%` }}
                  />
                </span>
              </button>
              {hasChildren && (
                <button
                  type="button"
                  aria-label={`${collapsed.has(span.id) ? "Expand" : "Collapse"} ${span.name}`}
                  onClick={() => handleToggle(span.id)}
                  className="absolute top-1/2 -translate-y-1/2 rounded-sm hover:bg-secondary"
                  style={{ left: depth * INDENT_PX + 6 }}
                >
                  {collapsed.has(span.id) ? <ChevronRight className="size-3" /> : <ChevronDown className="size-3" />}
                </button>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
