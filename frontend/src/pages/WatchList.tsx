import {
  addSymbolInPriceBoard,
  deleteWatchlistSymbol,
  getPriceBoard,
  getSectorPriceBoard,
  getSectors,
  getWatchlist,
} from "@/api/stock";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  PRICE_STATE,
  calculateChangePercent,
  formatExchange,
  formatPrice,
  formatVolume,
  getPriceState,
  type PriceState,
} from "@/lib/stock";
import type { PriceBoard, Sector, WatchlistEntry } from "@/types/stock";
import {
  ArrowDown,
  ArrowUp,
  ChevronsUpDown,
  EllipsisVertical,
  ListFilter,
  Plus,
  RotateCw,
  Star,
  Trash2,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";

type SectorCode = string | null;

type SortDirection = "asc" | "desc";
type SortState = { column: string; direction: SortDirection };

/* The three exchanges a listed Vietnamese ticker trades on, spelled the way
   formatExchange spells them - the price board's own HSX never reaches the
   table. All three selected is the same as no filter. */
const EXCHANGES = ["HOSE", "HNX", "UPCOM"];

type BoardColumn = {
  key: string;
  label: string;
  isNumeric: boolean;
  className?: string;
  value?: (stock: PriceBoard) => string | number;
};

/* Every column of the board, in render order, paired with the value it sorts on.
   The header row is generated from this; the body is not, because every cell
   renders differently, but the two have to stay in the same order. */
const COLUMNS: BoardColumn[] = [
  { key: "position", label: "#", isNumeric: true, className: "w-[52px] text-muted-foreground" },
  { key: "symbol", label: "Symbol", isNumeric: false, className: "w-[88px]", value: (s) => s.listingInfo.symbol },
  {
    key: "company",
    label: "Company",
    isNumeric: false,
    className: "min-w-[180px]",
    value: (s) => s.listingInfo.enOrganShortName,
  },
  { key: "exchange", label: "Exchange", isNumeric: false, className: "w-[104px]" },
  { key: "match", label: "Match", isNumeric: true, value: (s) => s.matchPrice.matchPrice },
  {
    key: "change",
    label: "Change %",
    isNumeric: true,
    // An untraded symbol has no change, and sorts as flat rather than as a fall
    value: (s) =>
      s.matchPrice.matchPrice > 0
        ? (s.matchPrice.matchPrice - s.matchPrice.referencePrice) / s.matchPrice.referencePrice
        : 0,
  },
  { key: "volume", label: "Volume", isNumeric: true, value: (s) => s.matchPrice.accumulatedVolume },
  { key: "ceiling", label: "Ceiling", isNumeric: true, className: "text-price-ceiling" },
  { key: "floor", label: "Floor", isNumeric: true, className: "text-price-floor" },
  { key: "reference", label: "Ref", isNumeric: true, className: "text-price-ref" },
  { key: "highest", label: "High", isNumeric: true },
  { key: "lowest", label: "Low", isNumeric: true },
  // No label: an icon column heading reads as a data column that lost its name
  { key: "actions", label: "", isNumeric: false, className: "w-[44px]" },
];

// Order two rows of the board by whichever column is sorted
const compareRows = (a: PriceBoard, b: PriceBoard, sort: SortState): number => {
  const column = COLUMNS.find((candidate) => candidate.key === sort.column);
  if (!column?.value) return 0;

  /* Numbers subtract, everything else compares as text. The old test was on
     both sides being strings, so one row with a missing name fell through to
     Number(undefined) - and a NaN does not just misplace its own row, it makes
     the comparator inconsistent and scrambles rows that were fine. Coercing to
     text sorts the gap to one end instead. Vietnamese names order by their
     accented letters rather than their code points, hence the locale. */
  const left = column.value(a);
  const right = column.value(b);
  const order =
    typeof left === "number" && typeof right === "number"
      ? left - right
      : String(left ?? "").localeCompare(String(right ?? ""), "vi");

  return sort.direction === "asc" ? order : -order;
};

function Header({ onSymbolAdded, onRefresh }: { onSymbolAdded: () => void; onRefresh: () => void }) {
  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });
  const onSubmit = async (data: { symbols: string }) => {
    try {
      await addSymbolInPriceBoard(data.symbols.toUpperCase());
    } catch (error) {
      console.log(error);
    } finally {
      form.reset();
      onSymbolAdded();
    }
  };

  return (
    <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-5 sm:px-6 lg:flex-row lg:items-end lg:justify-between lg:px-8">
        <div className="flex flex-col gap-1">
          <h1 className="text-xl font-bold tracking-tight">Your Smart Watchlist</h1>
          <p className="text-sm text-muted-foreground">Real-time AI insights for your selected tickers</p>
        </div>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex w-full gap-2 lg:w-auto">
            <FormField
              control={form.control}
              name="symbols"
              render={({ field }) => (
                <FormItem className="flex-1 lg:w-64">
                  <FormControl>
                    <Input
                      {...field}
                      aria-label="Stock symbol"
                      placeholder="Add a symbol, e.g. FPT"
                      className="uppercase placeholder:normal-case"
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button type="submit">
              <Plus aria-hidden="true" />
              <span className="hidden sm:inline">Add</span>
            </Button>
            <Button type="button" variant="outline" size="icon" onClick={onRefresh} aria-label="Refresh prices">
              <RotateCw aria-hidden="true" />
            </Button>
          </form>
        </Form>
      </div>
    </header>
  );
}

/* Favourites plus one button per ICB industry. A radio group rather than a row
   of independent buttons: exactly one is active at a time, so aria-pressed on
   plain buttons would understate the relationship for a screen reader. */
function SectorFilter({
  sectors,
  activeSector,
  onSelect,
}: {
  sectors: Sector[];
  activeSector: SectorCode;
  onSelect: (code: SectorCode) => void;
}) {
  return (
    <div role="radiogroup" aria-label="Filter by industry" className="flex flex-wrap gap-2">
      <Button
        role="radio"
        aria-checked={activeSector === null}
        variant={activeSector === null ? "default" : "outline"}
        size="sm"
        onClick={() => onSelect(null)}
      >
        <Star aria-hidden="true" className={activeSector === null ? "fill-current" : ""} />
        Favourites
      </Button>

      {sectors.map((sector) => (
        <Button
          key={sector.code}
          role="radio"
          aria-checked={activeSector === sector.code}
          variant={activeSector === sector.code ? "default" : "outline"}
          size="sm"
          disabled={sector.count === 0}
          onClick={() => onSelect(sector.code)}
        >
          {sector.name}
          <span className="text-xs opacity-60 tabular-nums">{sector.count}</span>
        </Button>
      ))}
    </div>
  );
}

/* The board's colour code, spelled out. Five hues that a new user cannot be
   expected to decode, and two pairs of them are near-identical in luminance. */
function BoardLegend() {
  const order: PriceState[] = ["ceiling", "up", "reference", "down", "floor"];
  return (
    <ul className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
      {order.map((state) => {
        const s = PRICE_STATE[state];
        return (
          <li key={state} className="flex items-center gap-1.5">
            <span aria-hidden="true" className={`${s.text} font-mono text-[10px] leading-none`}>
              {s.sign}
            </span>
            <span className={`${s.text} font-medium`}>{s.label}</span>
            <span className="text-muted-foreground">({s.vi})</span>
          </li>
        );
      })}
    </ul>
  );
}

/* A price cell: colour plus a sign, so the state survives a greyscale print,
   a colour-blind reader, and a bad monitor. A zero price means the symbol has
   not traded yet - routine across a whole industry, and especially on UPCOM -
   and renders as a dash, not as a full-red floor. */
function PriceCell({
  value,
  reference,
  ceiling,
  floor,
  showSign = true,
  className = "",
}: {
  value: number;
  reference: number;
  ceiling?: number;
  floor?: number;
  showSign?: boolean;
  className?: string;
}) {
  if (value <= 0) {
    return <TableCell className={`text-right font-mono tabular-nums text-muted-foreground ${className}`}>-</TableCell>;
  }

  const state = getPriceState({ price: value, reference, ceiling, floor });
  const s = PRICE_STATE[state];
  return (
    <TableCell className={`text-right font-mono tabular-nums ${s.text} ${className}`}>
      <span className="sr-only">{s.label}: </span>
      {showSign && (
        <span aria-hidden="true" className="mr-1 text-[10px]">
          {s.sign}
        </span>
      )}
      {formatPrice(value)}
    </TableCell>
  );
}

/* One column header. A column that can be ordered gets a real button rather
   than a click handler on the cell, so sorting is reachable by keyboard, and
   aria-sort so a screen reader is told which column is ordered and which way.
   A column with no sort value renders as plain text, with no control to press
   and nothing announced. */
function BoardHead({
  column,
  sort,
  onSort,
}: {
  column: BoardColumn;
  sort: SortState | null;
  onSort: (key: string) => void;
}) {
  const className = `font-semibold ${column.isNumeric ? "text-right" : ""} ${column.className ?? ""}`;

  if (!column.value) {
    return <TableHead className={className}>{column.label}</TableHead>;
  }

  const direction = sort?.column === column.key ? sort.direction : null;
  const Icon = direction === null ? ChevronsUpDown : direction === "asc" ? ArrowUp : ArrowDown;

  return (
    <TableHead
      aria-sort={direction === null ? "none" : direction === "asc" ? "ascending" : "descending"}
      className={className}
    >
      <button
        type="button"
        onClick={() => onSort(column.key)}
        className={`inline-flex w-full items-center gap-1 hover:opacity-80 ${column.isNumeric ? "justify-end" : ""}`}
      >
        {column.label}
        <Icon aria-hidden="true" className={`size-3 shrink-0 ${direction === null ? "opacity-40" : "opacity-100"}`} />
      </button>
    </TableHead>
  );
}

/* The Exchange header narrows the board instead of ordering it: with three
   possible values, "show me only HNX" is the question people actually have.
   Checkboxes rather than radios so two exchanges can be shown together, and the
   menu stays open while several are ticked. */
function ExchangeFilter({
  column,
  selected,
  onToggle,
}: {
  column: BoardColumn;
  selected: string[];
  onToggle: (exchange: string) => void;
}) {
  const isFiltered = selected.length < EXCHANGES.length;

  return (
    <TableHead className={`font-semibold ${column.className ?? ""}`}>
      <DropdownMenu>
        <DropdownMenuTrigger className="inline-flex w-full items-center gap-1 hover:opacity-80">
          {column.label}
          <ListFilter aria-hidden="true" className={`size-3 shrink-0 ${isFiltered ? "opacity-100" : "opacity-40"}`} />
          <span className="sr-only">
            {isFiltered ? `showing ${selected.join(", ") || "no exchange"}` : "showing every exchange"}
          </span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          {EXCHANGES.map((exchange) => (
            <DropdownMenuCheckboxItem
              key={exchange}
              checked={selected.includes(exchange)}
              onCheckedChange={() => onToggle(exchange)}
              onSelect={(event) => event.preventDefault()}
            >
              {exchange}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </TableHead>
  );
}

const WatchListPage = () => {
  const [priceBoard, setPriceBoard] = useState<PriceBoard[]>([]);
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [activeSector, setActiveSector] = useState<SectorCode>(null);
  const [sort, setSort] = useState<SortState | null>(null);
  const [exchanges, setExchanges] = useState<string[]>(EXCHANGES);
  const [watchlist, setWatchlist] = useState<WatchlistEntry[]>([]);
  const [pendingRemoval, setPendingRemoval] = useState<WatchlistEntry | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRemoving, setIsRemoving] = useState(false);

  /* Which tab the newest load was started for. Two loads run at once whenever a
     click lands before the previous board arrives, and the sector boards are the
     slow ones - without this the late response wins and the table ends up showing
     an industry other than the one the filter row has selected. */
  const requestedSector = useRef<SectorCode>(null);

  // Load the board for whichever tab is active
  const loadBoard = async (sector: SectorCode) => {
    requestedSector.current = sector;
    setIsLoading(true);
    try {
      const rows = sector === null ? await getPriceBoard() : await getSectorPriceBoard(sector);

      // Drop the response if the user has since picked another tab
      if (requestedSector.current !== sector) return;
      setPriceBoard(rows);
    } catch (error) {
      console.log(error);
      if (requestedSector.current === sector) setPriceBoard([]);
    } finally {
      if (requestedSector.current === sector) setIsLoading(false);
    }
  };

  /* The board comes from VietCap and carries no watchlist ids, so the entries
     are fetched alongside it - they are what says which rows can be removed, and
     by which id. A sector row that happens to be on the watchlist gets the same
     menu, since it is the same entry either way. */
  const loadWatchlist = async () => {
    try {
      setWatchlist(await getWatchlist());
    } catch (error) {
      console.log(error);
    }
  };

  // Remove the entry the dialog is confirming, then reload what depended on it
  const handleRemove = async () => {
    if (!pendingRemoval) return;

    setIsRemoving(true);
    try {
      await deleteWatchlistSymbol(pendingRemoval.id);

      // Only dismiss on success, so a failure leaves the dialog to try again
      setPendingRemoval(null);
      await Promise.all([loadWatchlist(), loadBoard(activeSector)]);
    } catch (error) {
      console.log(error);
    } finally {
      setIsRemoving(false);
    }
  };

  /* Each column cycles through three states: the board's own order, then
     descending, then ascending, then back to the board's order. The third state
     is the point - without it there is no way back to the order the backend
     chose, which for a sector board is most-active-first. */
  const handleSort = (key: string) => {
    setSort((current) => {
      if (current?.column !== key) return { column: key, direction: "desc" };
      return current.direction === "desc" ? { column: key, direction: "asc" } : null;
    });
  };

  // Ticking an exchange off hides its rows
  const handleExchangeToggle = (exchange: string) => {
    setExchanges((current) =>
      current.includes(exchange) ? current.filter((name) => name !== exchange) : [...current, exchange],
    );
  };

  // A new symbol always lands on the watchlist, so show it there
  const handleSymbolAdded = () => {
    loadWatchlist();
    if (activeSector === null) {
      loadBoard(null);
      return;
    }
    setActiveSector(null);
  };

  useEffect(() => {
    getSectors()
      .then(setSectors)
      .catch((error) => console.log(error));
    loadWatchlist();
  }, []);

  useEffect(() => {
    loadBoard(activeSector);
  }, [activeSector]);

  const activeSectorName = sectors.find((sector) => sector.code === activeSector)?.name;

  /* VietCap returns a row for every symbol it was asked about, and a row for a
     symbol it cannot resolve carries a null listingInfo. One bad ticker on the
     watchlist would otherwise take the whole page down through the router's
     error boundary. */
  /* Two guards in one pass. A row VietCap could not resolve carries a null
     listingInfo, and rendering it took the whole page down through the router's
     error boundary. A row whose symbol has already appeared is a duplicate, and
     two rows sharing a key let React drop or duplicate rows whenever the order
     changes - which is what made sorting by Symbol or Company scramble the
     table. Both are dropped here rather than guarded at every cell. */
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

  /* Three reasons the table can be empty, and they need different messages -
     "no listed tickers in this industry" is wrong and confusing when the rows
     are there and the exchange filter is what hid them. */
  const emptyMessage =
    rows.length > 0
      ? "No tickers on the selected exchanges."
      : activeSector === null
        ? "No symbols on your board yet. Add one above to start tracking it."
        : "No listed tickers in this industry.";

  // Say how the board is ordered, since a sort overrides the backend's order
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
      <Header onSymbolAdded={handleSymbolAdded} onRefresh={() => loadBoard(activeSector)} />

      <div className="mx-auto w-full max-w-7xl px-4 py-5 sm:px-6 lg:px-8">
        <SectorFilter sectors={sectors} activeSector={activeSector} onSelect={setActiveSector} />

        <div className="mt-4 mb-3 flex flex-wrap items-center justify-between gap-3">
          <BoardLegend />
          <p className="text-xs text-muted-foreground">{boardCaption}</p>
        </div>

        {/* overflow-x-auto on its own wrapper: ten numeric columns will not fit a
            phone, and the page body must never scroll sideways. */}
        <Dialog open={pendingRemoval !== null} onOpenChange={(open) => !open && setPendingRemoval(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Remove {pendingRemoval?.ticker} from your watchlist?</DialogTitle>
              <DialogDescription>
                It stops appearing on your Favourites board. Nothing about the ticker itself changes, and you can add
                it back at any time.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setPendingRemoval(null)} disabled={isRemoving}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleRemove} disabled={isRemoving}>
                <Trash2 aria-hidden="true" />
                {isRemoving ? "Removing..." : "Delete"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <div className="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  {COLUMNS.map((column) =>
                    column.key === "exchange" ? (
                      <ExchangeFilter
                        key={column.key}
                        column={column}
                        selected={exchanges}
                        onToggle={handleExchangeToggle}
                      />
                    ) : (
                      <BoardHead key={column.key} column={column} sort={sort} onSort={handleSort} />
                    ),
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={COLUMNS.length} className="h-32 text-center text-sm text-muted-foreground">
                      Loading prices...
                    </TableCell>
                  </TableRow>
                ) : visibleRows.length === 0 ? (
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={COLUMNS.length} className="h-32 text-center text-sm text-muted-foreground">
                      {emptyMessage}
                    </TableCell>
                  </TableRow>
                ) : (
                  visibleRows.map((stock: PriceBoard, position: number) => {
                    const { listingInfo, matchPrice } = stock;
                    const entry = watchlistByTicker.get(listingInfo.symbol);

                    // A symbol with no match yet has no change to report either
                    const hasTraded = matchPrice.matchPrice > 0;
                    const change = hasTraded
                      ? calculateChangePercent(matchPrice.matchPrice, matchPrice.referencePrice)
                      : null;

                    // Named arguments — the previous positional call passed
                    // (price, ceiling, floor, reference) into a function that
                    // expected (price, reference, ceiling, floor).
                    const priceArgs = {
                      reference: matchPrice.referencePrice,
                      ceiling: listingInfo.ceiling,
                      floor: listingInfo.floor,
                    };
                    const matchState = hasTraded
                      ? getPriceState({ price: matchPrice.matchPrice, ...priceArgs })
                      : "reference";
                    const matchStyle = PRICE_STATE[matchState];

                    return (
                      <TableRow key={listingInfo.symbol} className="transition-colors hover:bg-muted/50">
                        <TableCell className="text-right font-mono text-xs tabular-nums text-muted-foreground">
                          {position + 1}
                        </TableCell>
                        <TableCell className={`font-semibold ${matchStyle.text}`}>{listingInfo.symbol}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">{listingInfo.enOrganShortName}</TableCell>
                        <TableCell className="text-xs font-medium text-muted-foreground">
                          {formatExchange(listingInfo.board)}
                        </TableCell>

                        <PriceCell value={matchPrice.matchPrice} {...priceArgs} className="font-semibold" />

                        <TableCell className={`text-right font-mono font-semibold tabular-nums ${matchStyle.text}`}>
                          {change ? (
                            <>
                              <span className="sr-only">{matchStyle.label}: </span>
                              {change.isPositive ? "+" : ""}
                              {change.percent}%
                            </>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>

                        <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                          {formatVolume(matchPrice.accumulatedVolume)}
                        </TableCell>

                        {/* Ceiling, floor and reference are fixed bounds, not
                            movements, so they take their own colour and no arrow. */}
                        <TableCell className="text-right font-mono tabular-nums text-price-ceiling">
                          {formatPrice(listingInfo.ceiling)}
                        </TableCell>
                        <TableCell className="text-right font-mono tabular-nums text-price-floor">
                          {formatPrice(listingInfo.floor)}
                        </TableCell>
                        <TableCell className="text-right font-mono tabular-nums text-price-ref">
                          {formatPrice(matchPrice.referencePrice)}
                        </TableCell>

                        <PriceCell value={matchPrice.highest} {...priceArgs} showSign={false} />
                        <PriceCell value={matchPrice.lowest} {...priceArgs} showSign={false} />

                        {/* Only rows that are on the watchlist have anything to
                            act on, so a sector ticker you do not follow gets an
                            empty cell rather than a menu with a dead item. */}
                        <TableCell className="text-right">
                          {entry && (
                            <DropdownMenu>
                              <DropdownMenuTrigger
                                aria-label={`Actions for ${listingInfo.symbol}`}
                                className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                              >
                                <EllipsisVertical aria-hidden="true" className="size-4" />
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem variant="destructive" onSelect={() => setPendingRemoval(entry)}>
                                  <Trash2 aria-hidden="true" />
                                  Delete
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default WatchListPage;
