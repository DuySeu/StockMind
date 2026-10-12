// Span shapes streamed by POST /v1/chat when `trace` is on, and the tree the
// eval page renders from them. Mirrors internal/trace/trace.go.

export interface Span {
  id: string;
  parent_id?: string;
  kind: string;
  name: string;
  start: string;
  duration_ms: number;
  input?: unknown;
  output?: unknown;
  attrs?: Record<string, unknown>;
  error?: string;
  truncated?: boolean;
}

export type TraceEvent =
  | { phase: "start" | "end"; span: Span }
  | { phase: "summary"; dropped?: number };

export interface SpanNode {
  span: Span;
  startMs: number;
  isRunning: boolean;
  children: SpanNode[];
}

// Build the span tree from start/end events; an end copy replaces its start copy
export function buildSpanTree(events: readonly TraceEvent[]): SpanNode[] {
  // Keep the latest copy of each span
  const nodes = new Map<string, SpanNode>();
  for (const event of events) {
    if (event.phase === "summary") continue;
    const { span } = event;
    if (event.phase === "start" && nodes.has(span.id)) continue;
    nodes.set(span.id, {
      span,
      // Go sends nanoseconds; not every engine parses more than 3 fraction digits
      startMs: Date.parse(span.start.replace(/(\.\d{3})\d+/, "$1")),
      isRunning: event.phase === "start",
      children: [],
    });
  }

  // Attach children; a span whose parent never arrived (dropped) becomes a root
  const roots: SpanNode[] = [];
  for (const node of nodes.values()) {
    const parent = node.span.parent_id ? nodes.get(node.span.parent_id) : undefined;
    (parent ? parent.children : roots).push(node);
  }

  // Order siblings by start time
  const byStart = (a: SpanNode, b: SpanNode) => a.startMs - b.startMs;
  for (const node of nodes.values()) node.children.sort(byStart);
  return roots.sort(byStart);
}
