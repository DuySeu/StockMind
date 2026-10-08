import { errorMessage } from "@/api/sessions";
import {
  addSymbolInPriceBoard,
  deleteWatchlistSymbol,
  getMarketIndices,
  getPriceBoard,
  getSectorPriceBoard,
  getSectors,
  getWatchlist,
  moveWatchlistSymbol,
} from "@/api/stock";
import { PriceBoardTable } from "@/components/PriceBoardTable";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  COLUMNS,
  EXCHANGES,
  compareRows,
  isRequestCanceled,
  type SectorCode,
  type SortState,
} from "@/lib/priceBoard";
import {
  PRICE_STATE,
  formatExchange,
  formatIndexPoints,
  formatVolume,
  getPriceState,
} from "@/lib/stock";
import type { MarketIndex, PriceBoard, Sector, WatchlistEntry } from "@/types/stock";
import { Plus, RotateCw } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";

const INDEX_LABELS: Record<string, string> = {
  VNINDEX: "VN-Index",
  VN30: "VN30",
  VN100: "VN100",
};

// Render the three market index cards above the board
function IndexCards({ indices, isLoading }: { indices: MarketIndex[]; isLoading: boolean }) {
  if (indices.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">
        {isLoading ? "Loading the market indices..." : "The market indices are unavailable right now."}
      </p>
    );
  }

  return (
    <ul className="grid grid-cols-1 gap-2 sm:grid-cols-3">
      {indices.map((index) => {
        const state = PRICE_STATE[getPriceState({ price: index.value, reference: index.reference })];
        return (
          <li key={index.symbol}>
            <Card className="gap-1 p-3">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {INDEX_LABELS[index.symbol] ?? index.symbol}
                </span>
                <span className="font-mono text-[10px] tabular-nums text-muted-foreground">
                  {formatVolume(index.volume)}
                </span>
              </div>
              <p className={`font-mono text-lg font-semibold tabular-nums ${state.text}`}>
                {formatIndexPoints(index.value)}
              </p>
              <p className={`font-mono text-xs tabular-nums ${state.text}`}>
                <span className="sr-only">{state.label}: </span>
                <span aria-hidden="true" className="mr-1 text-[10px]">
                  {state.sign}
                </span>
                {index.change > 0 ? "+" : ""}
                {formatIndexPoints(index.change)} ({index.changePercent > 0 ? "+" : ""}
                {index.changePercent.toFixed(2)}%)
              </p>
            </Card>
          </li>
        );
      })}
    </ul>
  );
}

// Render the watchlist page with market indices and the price board
const WatchListPage = () => {
  const [priceBoard, setPriceBoard] = useState<PriceBoard[]>([]);
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [activeSector, setActiveSector] = useState<SectorCode>(null);
  const [sort, setSort] = useState<SortState | null>(null);
  const [exchanges, setExchanges] = useState<string[]>([...EXCHANGES]);
  const [watchlist, setWatchlist] = useState<WatchlistEntry[]>([]);
  const [indices, setIndices] = useState<MarketIndex[]>([]);
  const [pendingRemoval, setPendingRemoval] = useState<WatchlistEntry | null>(null);
  const [boardError, setBoardError] = useState<string | null>(null);
  const [reorderError, setReorderError] = useState<string | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isIndicesLoading, setIsIndicesLoading] = useState(true);
  const [isRemoving, setIsRemoving] = useState(false);

  const boardAbortRef = useRef<AbortController | null>(null);
  const indicesAbortRef = useRef<AbortController | null>(null);

  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });

  // Load the board for the active Favourites or sector tab
  const loadBoard = async (sector: SectorCode) => {
    boardAbortRef.current?.abort();
    const controller = new AbortController();
    boardAbortRef.current = controller;

    setIsLoading(true);
    setBoardError(null);
    try {
      const rows =
        sector === null
          ? await getPriceBoard(undefined, controller.signal)
          : await getSectorPriceBoard(sector, undefined, controller.signal);
      if (controller.signal.aborted) return;
      setPriceBoard(rows);
    } catch (error) {
      if (isRequestCanceled(error) || controller.signal.aborted) return;
      console.error(error);
      setPriceBoard([]);
      setBoardError(errorMessage(error, "Could not load prices."));
    } finally {
      if (!controller.signal.aborted) setIsLoading(false);
    }
  };

  // Load watchlist entries used for row actions
  const loadWatchlist = async () => {
    try {
      setWatchlist(await getWatchlist());
    } catch (error) {
      console.error(error);
      setBoardError(errorMessage(error, "Could not load your watchlist."));
    }
  };

  // Load VN-Index, VN30 and VN100
  const loadIndices = async () => {
    indicesAbortRef.current?.abort();
    const controller = new AbortController();
    indicesAbortRef.current = controller;

    setIsIndicesLoading(true);
    try {
      const next = await getMarketIndices(controller.signal);
      if (controller.signal.aborted) return;
      setIndices(next);
    } catch (error) {
      if (isRequestCanceled(error) || controller.signal.aborted) return;
      console.error(error);
      setIndices([]);
    } finally {
      if (!controller.signal.aborted) setIsIndicesLoading(false);
    }
  };

  // Add a ticker to the watchlist and report server validation on the field
  const handleAddSymbol = async (data: { symbols: string }) => {
    const symbol = data.symbols.trim().toUpperCase();
    if (!symbol) {
      form.setError("symbols", { message: "Type a ticker first, for example FPT." });
      return;
    }

    try {
      await addSymbolInPriceBoard(symbol);
    } catch (error) {
      form.setError("symbols", { message: errorMessage(error, `Could not add ${symbol} to your watchlist.`) });
      return;
    }

    form.reset();
    loadWatchlist();
    if (activeSector === null) {
      loadBoard(null);
      return;
    }
    setActiveSector(null);
  };

  // Remove the confirmed watchlist entry and reload dependent data
  const handleRemove = async () => {
    if (!pendingRemoval) return;

    setIsRemoving(true);
    setRemoveError(null);
    try {
      await deleteWatchlistSymbol(pendingRemoval.id);
      setPendingRemoval(null);
      await Promise.all([loadWatchlist(), loadBoard(activeSector)]);
    } catch (error) {
      console.error(error);
      setRemoveError(errorMessage(error, `Could not remove ${pendingRemoval.ticker}.`));
    } finally {
      setIsRemoving(false);
    }
  };

  // Move one watchlist row up or down, then reload
  const handleMove = async (entry: WatchlistEntry, direction: "up" | "down") => {
    setReorderError(null);
    try {
      await moveWatchlistSymbol(entry.id, direction);
      await Promise.all([loadWatchlist(), loadBoard(activeSector)]);
    } catch (error) {
      setReorderError(errorMessage(error, `Could not move ${entry.ticker}.`));
    }
  };

  // Cycle a column through desc → asc → backend order
  const handleSort = (key: string) => {
    setSort((current) => {
      if (current?.column !== key) return { column: key, direction: "desc" };
      return current.direction === "desc" ? { column: key, direction: "asc" } : null;
    });
  };

  // Toggle an exchange in the column filter
  const handleExchangeToggle = (exchange: string) => {
    setExchanges((current) =>
      current.includes(exchange) ? current.filter((name) => name !== exchange) : [...current, exchange],
    );
  };

  // Refresh board prices and market indices
  const handleRefresh = () => {
    loadBoard(activeSector);
    loadIndices();
  };

  useEffect(() => {
    const controller = new AbortController();

    getSectors(controller.signal)
      .then(setSectors)
      .catch((error) => {
        if (isRequestCanceled(error)) return;
        console.error(error);
        setBoardError(errorMessage(error, "Could not load industries."));
      });
    loadWatchlist();
    loadIndices();

    return () => {
      controller.abort();
      boardAbortRef.current?.abort();
      indicesAbortRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    loadBoard(activeSector);
  }, [activeSector]);

  const activeSectorName = sectors.find((sector) => sector.code === activeSector)?.name;
  const isFavourites = activeSector === null;
  const reorderBlockedBy = sort
    ? "Clear the column sort to reorder rows."
    : exchanges.length < EXCHANGES.length
      ? "Show every exchange to reorder rows."
      : null;

  // Drop unresolved or duplicate rows before exchange filter and sort
  const watchlistByTicker = new Map(watchlist.map((entry) => [entry.ticker, entry]));
  const shown = new Set<string>();
  const rows = priceBoard.filter((stock) => {
    if (!stock?.listingInfo || !stock?.matchPrice) return false;
    if (shown.has(stock.listingInfo.symbol)) return false;
    shown.add(stock.listingInfo.symbol);
    return true;
  });
  const onSelectedExchanges = rows.filter((stock) => exchanges.includes(formatExchange(stock.listingInfo.board)));
  const visibleRows = sort ? [...onSelectedExchanges].sort((a, b) => compareRows(a, b, sort)) : onSelectedExchanges;

  const emptyMessage =
    rows.length > 0
      ? "No tickers on the selected exchanges."
      : activeSector === null
        ? "No symbols on your board yet. Add one above to start tracking it."
        : "No listed tickers in this industry.";

  const sortedColumn = COLUMNS.find((column) => column.key === sort?.column);
  const sortedOrder = sortedColumn?.isNumeric
    ? sort?.direction === "asc"
      ? "lowest first"
      : "highest first"
    : sort?.direction === "asc"
      ? "A to Z"
      : "Z to A";
  const boardCaption = sortedColumn
    ? `Sorted by ${sortedColumn.label}, ${sortedOrder}`
    : activeSectorName
      ? `${activeSectorName} - most active first`
      : "Prices in thousands of VND";

  return (
    <div className="flex w-full flex-1 flex-col">
      <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-5 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div className="flex flex-col gap-1">
              <h1 className="text-xl font-bold tracking-tight">Your Smart Watchlist</h1>
              <p className="text-sm text-muted-foreground">Real-time AI insights for your selected tickers</p>
            </div>

            <Form {...form}>
              <form onSubmit={form.handleSubmit(handleAddSymbol)} className="flex w-full items-start gap-2 lg:w-auto">
                <FormField
                  control={form.control}
                  name="symbols"
                  render={({ field }) => (
                    <FormItem className="flex-1 lg:w-64">
                      <FormControl>
                        <Input
                          {...field}
                          aria-label="Stock symbol"
                          placeholder="Add a symbol to your watchlist"
                          className="uppercase placeholder:normal-case"
                          onChange={(event) => {
                            form.clearErrors("symbols");
                            field.onChange(event);
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type="submit">
                  <Plus aria-hidden="true" />
                  <span className="hidden sm:inline">Add</span>
                </Button>
                <Button type="button" variant="outline" size="icon" onClick={handleRefresh} aria-label="Refresh prices">
                  <RotateCw aria-hidden="true" />
                </Button>
              </form>
            </Form>
          </div>

          <IndexCards indices={indices} isLoading={isIndicesLoading} />
        </div>
      </header>

      <PriceBoardTable
        sectors={sectors}
        activeSector={activeSector}
        onSelectSector={setActiveSector}
        boardCaption={boardCaption}
        boardError={boardError}
        reorderError={reorderError}
        removeError={removeError}
        isLoading={isLoading}
        emptyMessage={emptyMessage}
        visibleRows={visibleRows}
        watchlistByTicker={watchlistByTicker}
        sort={sort}
        onSort={handleSort}
        exchanges={exchanges}
        onToggleExchange={handleExchangeToggle}
        isFavourites={isFavourites}
        reorderBlockedBy={reorderBlockedBy}
        onMove={handleMove}
        pendingRemoval={pendingRemoval}
        onPendingRemovalChange={(entry) => {
          setRemoveError(null);
          setPendingRemoval(entry);
        }}
        isRemoving={isRemoving}
        onConfirmRemove={handleRemove}
      />
    </div>
  );
};

export default WatchListPage;
