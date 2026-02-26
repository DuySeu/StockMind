import { getWatchlist, streamMarketResearch } from "@/api/stock";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Check, Loader2, Plus, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

interface ProgressStep {
  ticker: string;
  step: string;
  message: string;
  progress: number;
}

const stepLabels: Record<string, string> = {
  building_prompt: "Building research prompt",
  submitting: "Submitting to Tavily AI",
  polling: "Researching & gathering data",
  parsing: "Parsing results",
  completed: "Completed",
  failed: "Failed",
};

const MarketResearcherPage = () => {
  const navigate = useNavigate();
  const [watchList, setWatchList] = useState<any[]>([]);
  const [researchList, setResearchList] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    getWatchlist().then((response) => {
      setWatchList(response);
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
    <div className="flex flex-col items-center p-3 gap-4">
      <span className="text-sm text-accent mt-4 px-3 py-1 rounded-full bg-accent/10">
        Powered by Tavily <strong>/research</strong>
      </span>
      <div className="text-2xl font-bold text-primary">Market Researcher</div>
      <div className="text-sm text-muted-foreground">
        Get comprehensive market insights and analysis for your favorite stocks.
      </div>
      <div className="flex flex-col w-full max-w-4xl p-3 gap-4 bg-primary rounded-lg">
        <span className="text-xl text-center font-bold text-primary-foreground">Enter stock Tickers</span>
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
                      className="text-primary placeholder:text-primary"
                      placeholder="Enter symbols to research..."
                      disabled={isLoading}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button type="submit" size="icon" variant="secondary" disabled={isLoading}>
              <Plus />
            </Button>
          </form>
        </Form>
        <span className="text-primary-foreground">Watchlist stocks:</span>
        <div className="flex flex-wrap gap-2">
          {watchList.map((stock) => (
            <Button
              key={stock.id}
              variant="secondary"
              disabled={researchList.includes(stock.ticker) || researchList.length >= 5 || isLoading}
              onClick={() => setResearchList([...researchList, stock.ticker])}
            >
              {stock.ticker}
            </Button>
          ))}
        </div>
        {researchList.length > 0 && (
          <>
            <span className="text-primary-foreground">Selected tickers ({researchList.length}/5):</span>
            <div className="flex flex-wrap gap-2">
              {researchList.map((stock) => (
                <div key={stock} className="flex items-center bg-secondary rounded-md">
                  <span className="text-secondary-foreground p-2 pr-0">{stock}</span>
                  <Button
                    variant="secondary"
                    size="icon-sm"
                    disabled={isLoading}
                    onClick={() => setResearchList(researchList.filter((s) => s !== stock))}
                  >
                    <X />
                  </Button>
                </div>
              ))}
            </div>
          </>
        )}

        <div className="flex gap-2">
          <Button
            disabled={researchList.length === 0 || isLoading}
            variant="secondary"
            onClick={handleResearch}
            className="flex-1"
          >
            Get Daily Digest ({researchList.length} tickers)
          </Button>
          {isLoading && (
            <Button variant="destructive" onClick={handleCancel}>
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
                  <div
                    key={ticker}
                    className="flex items-center gap-3 p-3 rounded-lg bg-secondary/10 border border-border"
                  >
                    {/* Status icon */}
                    <div className="flex-shrink-0">
                      {isCompleted ? (
                        <div className="h-6 w-6 rounded-full bg-green-500/20 flex items-center justify-center">
                          <Check className="h-3.5 w-3.5 text-green-500" />
                        </div>
                      ) : isFailed ? (
                        <div className="h-6 w-6 rounded-full bg-red-500/20 flex items-center justify-center">
                          <span className="text-red-500 text-xs font-bold">!</span>
                        </div>
                      ) : (
                        <Loader2 className="h-6 w-6 text-accent animate-spin" />
                      )}
                    </div>

                    {/* Ticker info + progress bar */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-primary-foreground text-sm">{ticker}</span>
                        <span className="text-xs text-muted-foreground truncate">
                          {stepLabels[latestStep.step] ?? latestStep.step}
                        </span>
                      </div>
                      <Progress
                        value={latestStep.progress}
                        className={`h-1.5 mt-1.5 ${
                          isCompleted
                            ? "[&>[data-slot=progress-indicator]]:bg-green-500"
                            : isFailed
                              ? "[&>[data-slot=progress-indicator]]:bg-red-500"
                              : ""
                        }`}
                      />
                    </div>

                    {/* Percentage */}
                    <span className="text-xs text-muted-foreground flex-shrink-0">
                      {Math.round(latestStep.progress)}%
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default MarketResearcherPage;
