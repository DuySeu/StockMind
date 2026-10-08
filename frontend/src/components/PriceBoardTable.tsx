import { Button } from "@/components/ui/button";
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
  DropdownMenuLabel,
  DropdownMenuSeparator,
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
  Star,
  Trash2,
} from "lucide-react";
import {
  COLUMNS,
  EXCHANGES,
  type BoardColumn,
  type SectorCode,
  type SortState,
} from "@/lib/priceBoard";

type PriceBoardTableProps = {
  sectors: Sector[];
  activeSector: SectorCode;
  onSelectSector: (code: SectorCode) => void;
  boardCaption: string;
  boardError: string | null;
  reorderError: string | null;
  removeError: string | null;
  isLoading: boolean;
  emptyMessage: string;
  visibleRows: PriceBoard[];
  watchlistByTicker: Map<string, WatchlistEntry>;
  sort: SortState | null;
  onSort: (key: string) => void;
  exchanges: string[];
  onToggleExchange: (exchange: string) => void;
  isFavourites: boolean;
  reorderBlockedBy: string | null;
  onMove: (entry: WatchlistEntry, direction: "up" | "down") => void;
  pendingRemoval: WatchlistEntry | null;
  onPendingRemovalChange: (entry: WatchlistEntry | null) => void;
  isRemoving: boolean;
  onConfirmRemove: () => void;
};

// Render Favourites plus one button per ICB industry
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

// Render the five board price-state colours with labels
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

// Render a price cell coloured by ceiling / floor / reference state
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

// Render a sortable column header with aria-sort
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

// Render the exchange column header as a multi-select filter
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

// Render the sector filter, legend, and price board table
export function PriceBoardTable({
  sectors,
  activeSector,
  onSelectSector,
  boardCaption,
  boardError,
  reorderError,
  removeError,
  isLoading,
  emptyMessage,
  visibleRows,
  watchlistByTicker,
  sort,
  onSort,
  exchanges,
  onToggleExchange,
  isFavourites,
  reorderBlockedBy,
  onMove,
  pendingRemoval,
  onPendingRemovalChange,
  isRemoving,
  onConfirmRemove,
}: PriceBoardTableProps) {
  return (
    <div className="mx-auto w-full max-w-7xl px-4 py-5 sm:px-6 lg:px-8">
      <SectorFilter sectors={sectors} activeSector={activeSector} onSelect={onSelectSector} />

      <div className="mt-4 mb-3 flex flex-wrap items-center justify-between gap-3">
        <BoardLegend />
        <p className="text-xs text-muted-foreground">{boardCaption}</p>
      </div>

      {boardError && (
        <p role="alert" className="mb-3 text-sm text-destructive">
          {boardError}
        </p>
      )}

      {reorderError && (
        <p role="alert" className="mb-3 text-sm text-destructive">
          {reorderError}
        </p>
      )}

      {removeError && (
        <p role="alert" className="mb-3 text-sm text-destructive">
          {removeError}
        </p>
      )}

      <Dialog open={pendingRemoval !== null} onOpenChange={(open) => !open && onPendingRemovalChange(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove {pendingRemoval?.ticker} from your watchlist?</DialogTitle>
            <DialogDescription>
              It stops appearing on your Favourites board. Nothing about the ticker itself changes, and you can add it
              back at any time.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => onPendingRemovalChange(null)} disabled={isRemoving}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={onConfirmRemove} disabled={isRemoving}>
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
                      onToggle={onToggleExchange}
                    />
                  ) : (
                    <BoardHead key={column.key} column={column} sort={sort} onSort={onSort} />
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
                visibleRows.map((stock, position) => {
                  const { listingInfo, matchPrice } = stock;
                  const entry = watchlistByTicker.get(listingInfo.symbol);
                  const hasTraded = matchPrice.matchPrice > 0;
                  const change = hasTraded
                    ? calculateChangePercent(matchPrice.matchPrice, matchPrice.referencePrice)
                    : null;
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
                              {isFavourites && (
                                <>
                                  {reorderBlockedBy && (
                                    <DropdownMenuLabel className="max-w-[13rem] text-xs font-normal text-muted-foreground">
                                      {reorderBlockedBy}
                                    </DropdownMenuLabel>
                                  )}
                                  <DropdownMenuItem
                                    disabled={reorderBlockedBy !== null || position === 0}
                                    onSelect={() => onMove(entry, "up")}
                                  >
                                    <ArrowUp aria-hidden="true" />
                                    Move up
                                  </DropdownMenuItem>
                                  <DropdownMenuItem
                                    disabled={reorderBlockedBy !== null || position === visibleRows.length - 1}
                                    onSelect={() => onMove(entry, "down")}
                                  >
                                    <ArrowDown aria-hidden="true" />
                                    Move down
                                  </DropdownMenuItem>
                                  <DropdownMenuSeparator />
                                </>
                              )}
                              <DropdownMenuItem variant="destructive" onSelect={() => onPendingRemovalChange(entry)}>
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
  );
}
