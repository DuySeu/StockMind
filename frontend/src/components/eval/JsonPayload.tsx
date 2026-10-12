import { Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

interface Props {
  label: string;
  value: unknown;
}

// Pretty-print a payload, unwrapping strings that hold JSON (tool args and results)
export function JsonPayload({ label, value }: Props) {
  if (value === undefined || value === null) return null;

  let parsed = value;
  if (typeof value === "string") {
    try {
      parsed = JSON.parse(value);
    } catch {
      // Plain text, or JSON cut off by truncation: show it as is
    }
  }
  const text = typeof parsed === "string" ? parsed : JSON.stringify(parsed, null, 2);

  return (
    <section>
      <div className="mb-1 flex items-center justify-between">
        <h3 className="text-xs font-medium text-muted-foreground">{label}</h3>
        <Button
          variant="ghost"
          size="icon"
          aria-label={`Copy ${label.toLowerCase()}`}
          onClick={() => navigator.clipboard.writeText(text)}
        >
          <Copy className="size-3" />
        </Button>
      </div>
      <pre className="max-h-80 overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap break-all">{text}</pre>
    </section>
  );
}
