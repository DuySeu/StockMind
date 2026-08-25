import { getResearchReport, getResearchReportById, getWatchlist, streamMarketResearch } from "@/api/stock";
import ResearchReport from "@/components/containers/ResearchReport";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PRICE_STATE, calculateChangePercent, getPriceState } from "@/lib/stock";
import { Check, FileText, Loader2, Plus, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

const stepLabels: Record<string, string> = {
  building_prompt: "Building research prompt",
  submitting: "Submitting to Tavily AI",
  polling: "Researching & gathering data",
  parsing: "Parsing results",
  completed: "Completed",
  failed: "Failed",
};

interface ProgressStep {
  ticker: string;
  step: string;
  message: string;
  progress: number;
}

// Render a report icon that fetches and shows one stored report in a dialog
const ReportDialog = ({ reportId }: { reportId: string }) => {
  const [open, setOpen] = useState(false);
  const [reportData, setReportData] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const fetchReport = async () => {
    setLoading(true);
    setReportData(null);
    setOpen(true);
    try {
      const data = await getResearchReportById(reportId);
      setReportData(data);
    } catch (error) {
      console.error("Failed to fetch report:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <button
        type="button"
        aria-label="Open research report"
        className="cursor-pointer rounded-md p-1 text-muted-foreground transition-colors hover:text-primary"
        onClick={fetchReport}
      >
        <FileText className="size-4" aria-hidden="true" />
      </button>
      <DialogContent className="w-full sm:max-w-6xl">
        <DialogHeader className="text-primary">
          <DialogTitle>Research Report</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col items-center p-3 gap-4 overflow-y-auto no-scrollbar max-h-[50vh]">
          {loading ? (
            <div className="flex items-center justify-center p-10">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          ) : (
            <ResearchReport data={reportData} />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

// Render the ticker picker, live research progress, and past report table
const MarketResearcherPage = () => {
  const navigate = useNavigate();
  const [watchList, setWatchList] = useState<any[]>([]);
  const [researchList, setResearchList] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const abortRef = useRef<AbortController | null>(null);
  const [researchReport, setResearchReport] = useState<any[]>([]);

  useEffect(() => {
    getWatchlist().then((response) => {
      setWatchList(response);
    });
  }, []);

  useEffect(() => {
    getResearchReport().then((response) => {
      setResearchReport(response);
    });
  }, []);

  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });

  const onSubmit = (data: any) => {
    form.reset();
    if (researchList.length >= 5) return;
    setResearchList([...researchList, data.symbols.toUpperCase()]);
  };

  const handleResearch = () => {
    setIsLoading(true);
    setProgressSteps([]);

    const controller = streamMarketResearch(researchList, "mini", {
      onProgress: (event) => {
        setProgressSteps((prev) => {
          const existingIdx = prev.findIndex((s) => s.ticker === event.ticker && s.step === event.step);
          if (existingIdx >= 0) {
            const updated = [...prev];
            updated[existingIdx] = event;
            return updated;
          }
          return [...prev, event];
        });
      },
      onComplete: (data) => {
        setIsLoading(false);
        abortRef.current = null;
        navigate(`/research/${researchList[0].toLowerCase()}`, {
          state: { response: data },
        });
      },
      onError: (error) => {
        console.error("Stream error:", error);
        setIsLoading(false);
        abortRef.current = null;
      },
    });

    abortRef.current = controller;
  };

  const handleCancel = () => {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setIsLoading(false);
    setProgressSteps([]);
  };

  // Group steps by ticker, keeping only the latest step per ticker
  const tickerLatestSteps = Object.entries(
    progressSteps.reduce(
      (acc, step) => {
        acc[step.ticker] = step;
        return acc;
      },
      {} as Record<string, ProgressStep>,
    ),
  );

  return (
    <div className="flex w-full flex-1 flex-col">
      {/* Same header bar as the watchlist and document pages: a bordered strip
          with the title left and no centred stack, so the three data pages do
          not each introduce their own page shell. */}
      <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl flex-col gap-1 px-4 py-5 sm:px-6 lg:px-8">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-xl font-bold tracking-tight">Market Researcher</h1>
            <span className="rounded-md border border-border bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
              Powered by Tavily <span className="font-mono">/research</span>
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Deep-dive research on up to five tickers, cross-checked against live prices.
          </p>
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8">
        <div className="glass flex flex-col gap-4 rounded-xl p-5">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Enter stock tickers</h2>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2">
            <FormField
              control={form.control}
              name="symbols"
              render={({ field }) => (
                <FormItem className="flex-1">
                  <FormControl>
                    <Input
                      {...field}
                      aria-label="Stock symbol"
                      placeholder="Add a ticker, e.g. FPT"
                      className="uppercase placeholder:normal-case"
                      disabled={isLoading}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button type="submit" size="icon" disabled={isLoading} aria-label="Add ticker">
              <Plus aria-hidden="true" />
            </Button>
          </form>
        </Form>
        {/* Hidden when the board is empty — the old label stood over nothing.
            text-primary-foreground was also white-on-light: that token is the
            colour that sits ON primary, not a body colour. */}
        {watchList.length > 0 && (
          <>
            <span className="text-sm font-medium text-foreground">From your watchlist</span>
            <div className="flex flex-wrap gap-2">
              {watchList.map((stock) => (
                <Button
                  key={stock.id}
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={researchList.includes(stock.ticker) || researchList.length >= 5 || isLoading}
                  onClick={() => setResearchList([...researchList, stock.ticker])}
                >
                  {stock.ticker}
                </Button>
              ))}
            </div>
          </>
        )}
        {researchList.length > 0 && (
          <>
            <span className="text-sm font-medium text-foreground">
              Selected tickers <span className="font-mono tabular-nums text-muted-foreground">({researchList.length}/5)</span>
            </span>
            <div className="flex flex-wrap gap-2">
              {researchList.map((stock) => (
                <span
                  key={stock}
                  className="flex items-center gap-1 rounded-md bg-primary py-1 pl-3 pr-1 text-sm font-semibold text-primary-foreground"
                >
                  {stock}
                  {/* Ghost, not another filled button: a default-variant button
                      inside a primary chip painted primary-on-primary. */}
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Remove ${stock}`}
                    disabled={isLoading}
                    className="size-6 text-primary-foreground hover:bg-primary-foreground/20 hover:text-primary-foreground"
                    onClick={() => setResearchList(researchList.filter((s) => s !== stock))}
                  >
                    <X aria-hidden="true" />
                  </Button>
                </span>
              ))}
            </div>
          </>
        )}

        <div className="flex gap-2">
          <Button
            type="button"
            disabled={researchList.length === 0 || isLoading}
            onClick={handleResearch}
            className="flex-1"
          >
            {isLoading ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Researching…
              </>
            ) : (
              <>
                Run digest
                <span className="font-mono tabular-nums opacity-80">({researchList.length})</span>
              </>
            )}
          </Button>
          {/* Cancel stops a request; it destroys nothing, so it is not destructive. */}
          {isLoading && (
            <Button type="button" variant="outline" onClick={handleCancel}>
              Cancel
            </Button>
          )}
        </div>

        {/* Progress Section */}
        {isLoading && (
          <div className="space-y-4 animate-in fade-in duration-300">
            {/* Per-ticker progress */}
            <div className="space-y-3">
              {tickerLatestSteps.map(([ticker, latestStep]) => {
                const isCompleted = latestStep.step === "completed";
                const isFailed = latestStep.step === "failed";
                return (
                  /* Rows enter from the top because that is where they are
                     appended. No stagger: they arrive on the stream's own
                     schedule, which already separates them. */
                  <div
                    key={ticker}
                    className="flex items-center gap-3 rounded-lg border border-border bg-muted/40 p-3 animate-in fade-in slide-in-from-top-1 duration-200 ease-out"
                  >
                    {/* Status icon */}
                    <div className="shrink-0">
                      {isCompleted ? (
                        <span className="flex size-6 items-center justify-center rounded-full bg-status-ok-bg animate-in fade-in zoom-in-95 duration-200 ease-out">
                          <Check className="size-3.5 text-status-ok" aria-hidden="true" />
                        </span>
                      ) : isFailed ? (
                        <span className="flex size-6 items-center justify-center rounded-full bg-status-error-bg">
                          <span className="text-xs font-bold text-status-error" aria-hidden="true">
                            !
                          </span>
                        </span>
                      ) : (
                        <Loader2 className="size-6 animate-spin text-primary" aria-hidden="true" />
                      )}
                    </div>

                    {/* Ticker info + progress bar */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold">{ticker}</span>
                        <span className="truncate text-xs text-muted-foreground">
                          {stepLabels[latestStep.step] ?? latestStep.step}
                        </span>
                      </div>
                      <Progress
                        value={latestStep.progress}
                        aria-label={`${ticker} research progress`}
                        className={`mt-1.5 h-1.5 ${
                          isCompleted
                            ? "[&>[data-slot=progress-indicator]]:bg-status-ok"
                            : isFailed
                              ? "[&>[data-slot=progress-indicator]]:bg-status-error"
                              : ""
                        }`}
                      />
                    </div>

                    {/* Percentage */}
                    <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
                      {Math.round(latestStep.progress)}%
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
        {researchReport && researchReport.length > 0 ? (
          <section aria-labelledby="past-reports" className="overflow-hidden rounded-lg border border-border bg-card">
            <h2 id="past-reports" className="sr-only">
              Past research reports
            </h2>
            {/* Seven columns will not fit a phone; the scroll belongs to the
                table, never to the page body. */}
            <div className="overflow-x-auto">
              <Table>
                <TableCaption className="px-4 pb-4">
                  Every ticker Tavily recommended, checked against its price today.
                </TableCaption>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="font-semibold">Symbol</TableHead>
                    <TableHead className="text-right font-semibold">Reference</TableHead>
                    <TableHead className="font-semibold">Recommendation</TableHead>
                    <TableHead className="font-semibold">Generated</TableHead>
                    <TableHead className="text-right font-semibold">Latest</TableHead>
                    <TableHead className="text-right font-semibold">Change %</TableHead>
                    <TableHead className="text-center font-semibold">Report</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {researchReport.map((stock: any) => {
                    const change = calculateChangePercent(stock.price, stock.reference_price);
                    const state = getPriceState({ price: stock.price, reference: stock.reference_price });
                    const style = PRICE_STATE[state];
                    return (
                      <TableRow key={stock.id} className="transition-colors hover:bg-muted/50">
                        <TableCell className={`font-semibold ${style.text}`}>{stock.ticker}</TableCell>
                        <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                          {stock.reference_price}
                        </TableCell>
                        <TableCell className="text-sm font-semibold">{stock.recommendation.toUpperCase()}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {new Date(stock.created_at).toLocaleDateString("vi-VN")}
                        </TableCell>
                        <TableCell className={`text-right font-mono tabular-nums ${style.text}`}>
                          {stock.price}
                        </TableCell>
                        {/* Arrow as well as hue, matching the watchlist board. */}
                        <TableCell className={`text-right font-mono font-semibold tabular-nums ${style.text}`}>
                          <span className="sr-only">{style.label}: </span>
                          <span aria-hidden="true" className="mr-1 text-[10px]">
                            {style.sign}
                          </span>
                          {change.isPositive ? "+" : ""}
                          {change.percent}%
                        </TableCell>
                        <TableCell className="text-center">
                          <ReportDialog reportId={stock.id} />
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </section>
        ) : (
          <p className="rounded-lg border border-dashed border-border px-4 py-10 text-center text-sm text-muted-foreground">
            No reports yet. Pick up to five tickers above and run a digest to see them here.
          </p>
        )}
      </div>
    </div>
  );
};

export default MarketResearcherPage;
