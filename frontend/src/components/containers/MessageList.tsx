import type { Message, ToolCallContent } from "@/types/message";
import { isImage, isQuotaError, isThinking, isToolCall, isTurnError } from "@/types/message";
import { BatteryWarning, Brain, Check, LoaderCircle, RotateCcw, TriangleAlert, Workflow, Wrench, X } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

const markdownComponents = {
  table: (props: React.ComponentProps<"table">) => (
    <div className="my-3 overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm border-collapse" {...props} />
    </div>
  ),
  thead: (props: React.ComponentProps<"thead">) => (
    <thead className="bg-muted/60 text-left" {...props} />
  ),
  th: (props: React.ComponentProps<"th">) => (
    <th className="px-3 py-2 font-semibold border-b border-border whitespace-nowrap" {...props} />
  ),
  td: (props: React.ComponentProps<"td">) => (
    <td className="px-3 py-1.5 border-b border-border/50 tabular-nums" {...props} />
  ),
  tr: (props: React.ComponentProps<"tr">) => (
    <tr className="even:bg-muted/30" {...props} />
  ),
  code: ({ className, children, ...props }: React.ComponentProps<"code"> & { inline?: boolean }) => {
    const isBlock = className?.startsWith("language-");
    if (isBlock) {
      return (
        <pre className="my-2 overflow-x-auto rounded-lg bg-muted/60 p-3 text-xs leading-relaxed">
          <code className={className} {...props}>{children}</code>
        </pre>
      );
    }
    return (
      <code className="rounded bg-muted/60 px-1.5 py-0.5 text-[0.85em] font-mono" {...props}>
        {children}
      </code>
    );
  },
  ul: (props: React.ComponentProps<"ul">) => (
    <ul className="my-1 ml-4 list-disc space-y-0.5" {...props} />
  ),
  ol: (props: React.ComponentProps<"ol">) => (
    <ol className="my-1 ml-4 list-decimal space-y-0.5" {...props} />
  ),
  p: (props: React.ComponentProps<"p">) => (
    <p className="my-1.5 leading-relaxed" {...props} />
  ),
  strong: (props: React.ComponentProps<"strong">) => (
    <strong className="font-semibold" {...props} />
  ),
  h1: (props: React.ComponentProps<"h1">) => <h1 className="text-lg font-bold mt-4 mb-2" {...props} />,
  h2: (props: React.ComponentProps<"h2">) => <h2 className="text-base font-bold mt-3 mb-1.5" {...props} />,
  h3: (props: React.ComponentProps<"h3">) => <h3 className="text-sm font-bold mt-2 mb-1" {...props} />,
};

/** Icon + colour for a single tool's outcome. */
const toolStatusStyle = {
  running: {
    Icon: LoaderCircle,
    iconClass: "size-3 shrink-0 animate-spin",
    chipClass: "border-border bg-muted text-muted-foreground",
    label: "running",
  },
  success: {
    Icon: Check,
    iconClass: "size-3 shrink-0",
    // Explicit greens rather than a theme token: `success` is not in the
    // palette, and the chip has to read the same in light and dark.
    chipClass:
      "border-emerald-600/30 bg-emerald-500/10 text-emerald-800 dark:border-emerald-400/30 dark:text-emerald-300",
    label: "succeeded",
  },
  error: {
    Icon: X,
    iconClass: "size-3 shrink-0",
    chipClass:
      "border-red-600/30 bg-red-500/10 text-red-800 dark:border-red-400/30 dark:text-red-300",
    label: "failed",
  },
} as const;

/**
 * Panel styling for a failed turn. Red is for something that went wrong; amber is
 * for a spent quota, which is a limit being enforced rather than a fault — and
 * the same reason it never raises a toast. Explicit palette colours rather than
 * theme tokens, matching the tool chips above: neither shade is in the palette
 * and both have to hold up in light and dark.
 */
const turnErrorStyle = {
  failure: {
    Icon: TriangleAlert,
    panelClass: "border-red-600/30 bg-red-500/10",
    textClass: "text-red-800 dark:text-red-300",
    buttonClass:
      "border-red-600/40 text-red-800 hover:bg-red-500/15 dark:border-red-400/40 dark:text-red-300",
    label: "Error",
  },
  quota: {
    Icon: BatteryWarning,
    panelClass: "border-amber-600/30 bg-amber-500/10",
    textClass: "text-amber-800 dark:text-amber-300",
    buttonClass:
      "border-amber-600/40 text-amber-800 hover:bg-amber-500/15 dark:border-amber-400/40 dark:text-amber-300",
    label: "Quota exhausted",
  },
} as const;

/**
 * A max-mode turn streams its pipeline steps down the same `tool_call` channel as
 * the model's own tools, namespaced by id. Without telling them apart, an agent
 * step (`market_data`) rendered identically to a tool (`get_stock_price`) and the
 * run summary counted four steps as four extra tools.
 */
const STEP_ID_PREFIX = "step:";
const isPipelineStep = (tc: ToolCallContent) => tc.id.startsWith(STEP_ID_PREFIX);

interface MessageListProps {
  messages: Message[];
  /** Re-sends the question that produced a failed turn. Omit to hide Retry. */
  onRetry?: (content: string) => void;
  /** Retry is pointless while another turn is already streaming. */
  retryDisabled?: boolean;
}

const MessageList = ({ messages, onRetry, retryDisabled }: MessageListProps) => {
  return messages.map((message, index) => {
    const isUser = message.role === "user";

    // Extract metadata parts. Discriminated on `type`, so a streamed entry and
    // a replayed one are indistinguishable here.
    const images = message.metadata?.filter(isImage) ?? [];
    const thinkingBlocks = message.metadata?.filter(isThinking) ?? [];
    const toolCalls = message.metadata?.filter(isToolCall) ?? [];

    const turnErrors = message.metadata?.filter(isTurnError) ?? [];

    const hasToolCalls = toolCalls.length > 0;
    const runningCount = toolCalls.filter((t) => t.status === "running").length;
    const failedCount = toolCalls.filter((t) => t.status === "error").length;
    const stepCount = toolCalls.filter(isPipelineStep).length;
    const plainToolCount = toolCalls.length - stepCount;

    // The question this turn was answering, so Retry has something to re-send.
    const previous = index > 0 ? messages[index - 1] : undefined;
    const retryContent = previous?.role === "user" ? previous.content : undefined;
    // Driven by the tools' own statuses now, not by "assistant hasn't written
    // any text yet" — that heuristic reported success for tools that had failed.
    const isUsingTool = runningCount > 0;

    return (
      <div key={index} className={`flex gap-4 ${isUser ? "flex-row-reverse" : "flex-row"} items-end`}>
        <div className={`flex flex-col max-w-[85%] ${isUser ? "items-end" : "items-start"}`}>
          {/* Image attachments (user messages) */}
          {images.map((img, idx) => (
            <div key={`img-${idx}`} className="mb-2 max-w-full">
              <img
                src={img.image_url?.url}
                alt="Attachment"
                className="max-w-full h-auto rounded-lg border border-border shadow-sm max-h-64 object-contain inline-block bg-background"
              />
            </div>
          ))}

          {/* Thinking blocks (assistant messages) */}
          {thinkingBlocks.map((block, idx) => (
            <div key={`think-${idx}`} className="mb-2 w-full">
              <details open={block.is_open} className="glass group overflow-hidden rounded-lg">
                <summary className="flex cursor-pointer select-none items-center gap-2 px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring">
                  <Brain className="size-3.5 shrink-0" aria-hidden="true" />
                  <span>Thought process</span>
                </summary>
                {/* Full muted-foreground, not /80: at 80% over the page
                    background this text dropped under 4.5:1. */}
                <div className="border-t border-border p-3 text-xs leading-relaxed text-muted-foreground">
                  <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
                    {block.thinking}
                  </Markdown>
                </div>
              </details>
            </div>
          ))}

          {/* Tool run summary */}
          {hasToolCalls && (
            <span
              className={`mb-1 flex items-center gap-1.5 px-1 text-xs font-medium ${
                isUsingTool
                  ? "text-muted-foreground"
                  : failedCount > 0
                    ? "text-red-700 dark:text-red-400"
                    : "text-primary"
              }`}
            >
              {isUsingTool ? (
                <>
                  <LoaderCircle className="size-3.5 shrink-0 animate-spin" aria-hidden="true" />
                  {stepCount > 0 ? "Running pipeline…" : `Using ${runningCount > 1 ? `${runningCount} tools` : "tool"}…`}
                </>
              ) : failedCount > 0 ? (
                <>
                  <X className="size-3.5 shrink-0" aria-hidden="true" />
                  {failedCount} of {toolCalls.length} {stepCount > 0 ? "actions" : toolCalls.length > 1 ? "tools" : "tool"}{" "}
                  failed
                </>
              ) : stepCount > 0 ? (
                <>
                  <Check className="size-3.5 shrink-0" aria-hidden="true" />
                  {stepCount} agent {stepCount > 1 ? "steps" : "step"}
                  {plainToolCount > 0 && `, ${plainToolCount} ${plainToolCount > 1 ? "tools" : "tool"}`}
                </>
              ) : (
                <>
                  <Check className="size-3.5 shrink-0" aria-hidden="true" />
                  {toolCalls.length > 1 ? `${toolCalls.length} tools used` : "Tool used"}
                </>
              )}
            </span>
          )}

          {/* Tool call details — chips in call order, so a run of three tools
              reads as three discrete steps rather than a paragraph of grey text.
              Each carries its own outcome: a failed tool must not look like a
              successful one. */}
          {hasToolCalls && (
            <div className="mb-1.5 flex w-full flex-col gap-1">
              <div className="flex flex-wrap gap-1.5">
                {toolCalls.map((tc, idx) => {
                  const { Icon, iconClass, chipClass, label } = toolStatusStyle[tc.status];
                  const isStep = isPipelineStep(tc);
                  const Kind = isStep ? Workflow : Wrench;
                  return (
                    <span
                      key={tc.id || `tc-${idx}`}
                      className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs ${chipClass}`}
                      title={tc.arguments ? `${tc.name}(${tc.arguments})` : tc.name}
                    >
                      <Kind className="size-3 shrink-0 opacity-60" aria-hidden="true" />
                      {/* A step is an agent, not a callable — monospace would read
                          as a function name. */}
                      <span className={isStep ? "font-medium" : "font-mono"}>{tc.name}</span>
                      <Icon className={iconClass} aria-hidden="true" />
                      {/* Colour alone can't carry the outcome. */}
                      <span className="sr-only">
                        {isStep ? "agent step" : "tool"} {label}
                      </span>
                    </span>
                  );
                })}
              </div>

              {/* Why a tool failed is the one tool output worth showing inline —
                  it's short, and it explains an answer that would otherwise look
                  arbitrarily incomplete. */}
              {toolCalls
                .filter((tc) => tc.status === "error" && tc.result)
                .map((tc, idx) => (
                  <p
                    key={`tc-err-${tc.id || idx}`}
                    className="px-1 text-xs leading-relaxed text-red-700 dark:text-red-400"
                  >
                    <span className="font-mono">{tc.name}</span>: {tc.result}
                  </p>
                ))}
            </div>
          )}

          {/* Turn failure — rendered where the reply would have been, so the
              question never sits there looking merely unanswered. A spent quota
              is the one failure shown *only* here (no toast), so it carries its
              own amber styling: nothing crashed, the account simply has nothing
              left to spend. */}
          {turnErrors.map((failure, idx) => {
            const style = isQuotaError(failure.code) ? turnErrorStyle.quota : turnErrorStyle.failure;
            const { Icon } = style;
            return (
              <div
                key={`err-${idx}`}
                className={`mb-1.5 flex w-full flex-col items-start gap-2 rounded-lg border px-3 py-2.5 ${style.panelClass}`}
              >
                <p className={`flex items-start gap-2 text-sm leading-relaxed ${style.textClass}`}>
                  <Icon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                  <span>
                    <span className="sr-only">{style.label}: </span>
                    {failure.message}
                  </span>
                </p>
                {onRetry && retryContent && (
                  <button
                    type="button"
                    onClick={() => onRetry(retryContent)}
                    disabled={retryDisabled}
                    className={`inline-flex min-h-8 cursor-pointer items-center gap-1.5 rounded-md border px-2.5 text-xs font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-50 ${style.buttonClass}`}
                  >
                    <RotateCcw className="size-3.5 shrink-0" aria-hidden="true" />
                    Try again
                  </button>
                )}
              </div>
            );
          })}

          {/* Main text content */}
          {message.content ? (
            <div
              className={`text-sm leading-relaxed md:text-[15px] ${
                isUser
                  ? "rounded-xl rounded-tr-sm bg-primary px-4 py-2.5 text-primary-foreground shadow-xs"
                  : "text-foreground"
              }`}
            >
              <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
                {message.content}
              </Markdown>
            </div>
          ) : (
            // Show loading indicator for empty assistant messages (still
            // streaming). A turn that already failed is not still streaming.
            !isUser &&
            turnErrors.length === 0 && (
              <div className="flex items-center gap-2 py-1" role="status" aria-label="Generating response">
                <span className="flex items-center gap-1" aria-hidden="true">
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:0ms]" />
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:150ms]" />
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:300ms]" />
                </span>
                <span className="text-xs text-muted-foreground">Thinking…</span>
              </div>
            )
          )}
        </div>
      </div>
    );
  });
};

export default MessageList;
