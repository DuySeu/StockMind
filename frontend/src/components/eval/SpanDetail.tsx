import { Badge } from "@/components/ui/badge";
import { JsonPayload } from "@/components/eval/JsonPayload";
import type { SpanNode } from "@/lib/trace";

interface Props {
  node: SpanNode | undefined;
}

// Show one span: timing, status, attrs, and its input and output as JSON
export function SpanDetail({ node }: Props) {
  if (!node) {
    return <p className="p-4 text-sm text-muted-foreground">Select a span to see its details.</p>;
  }

  const { span } = node;
  const status = node.isRunning ? "running" : span.error ? "error" : "ok";
  const attrs = Object.entries(span.attrs ?? {});

  return (
    <div className="space-y-4 p-4 text-sm">
      <div>
        <h2 className="font-mono font-semibold break-all">{span.name}</h2>
        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="secondary">{span.kind}</Badge>
          <Badge variant={status === "error" ? "destructive" : "outline"}>{status}</Badge>
          <span className="font-mono">{node.isRunning ? "..." : `${span.duration_ms} ms`}</span>
          {span.truncated && <span>payload truncated at 8 KB</span>}
        </div>
      </div>

      {span.error && <p className="rounded-md bg-destructive/10 p-2 font-mono text-xs text-destructive">{span.error}</p>}

      {attrs.length > 0 && (
        <section>
          <h3 className="mb-1 text-xs font-medium text-muted-foreground">Attrs</h3>
          <dl className="grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 font-mono text-xs">
            {attrs.map(([key, value]) => (
              <div key={key} className="contents">
                <dt className="text-muted-foreground">{key}</dt>
                <dd className="break-all">{typeof value === "string" ? value : JSON.stringify(value)}</dd>
              </div>
            ))}
          </dl>
        </section>
      )}

      <JsonPayload label="Input" value={span.input} />
      <JsonPayload label="Output" value={span.output} />
    </div>
  );
}
