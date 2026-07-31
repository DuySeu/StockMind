import { addSymbolInPriceBoard, getPriceBoard } from "@/api/stock";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  PRICE_STATE,
  calculateChangePercent,
  formatNumber,
  formatPrice,
  getPriceState,
  type PriceState,
} from "@/lib/stock";
import type { PriceBoard } from "@/types/stock";
import { Plus, RotateCw } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";

function Header({ setPriceBoard }: { setPriceBoard: (priceBoard: PriceBoard[]) => void }) {
  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });
  const onSubmit = async (data: any) => {
    try {
      await addSymbolInPriceBoard(data.symbols.toUpperCase());
    } catch (error) {
      console.log(error);
    } finally {
      const res = await getPriceBoard();
      setPriceBoard(res);
      form.reset();
    }
  };

  const handleRefresh = async () => {
    const res = await getPriceBoard();
    setPriceBoard(res);
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
            <Button type="button" variant="outline" size="icon" onClick={handleRefresh} aria-label="Refresh prices">
              <RotateCw aria-hidden="true" />
            </Button>
          </form>
        </Form>
      </div>
    </header>
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
            {/* Full opacity: at /70 this dropped to 3.05:1 in light mode. */}
            <span className="text-muted-foreground">({s.vi})</span>
          </li>
        );
      })}
    </ul>
  );
}

/* A price cell: colour plus a sign, so the state survives a greyscale print,
   a colour-blind reader, and a bad monitor. */
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

const WatchListPage = () => {
  const [priceBoard, setPriceBoard] = useState<PriceBoard[]>([]);

  useEffect(() => {
    getPriceBoard().then((res) => {
      setPriceBoard(res);
    });
  }, []);

  return (
    <div className="flex w-full flex-1 flex-col">
      <Header setPriceBoard={setPriceBoard} />

      <div className="mx-auto w-full max-w-7xl px-4 py-5 sm:px-6 lg:px-8">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <BoardLegend />
          <p className="text-xs text-muted-foreground">
            Prices in thousands of VND
          </p>
        </div>

        {/* overflow-x-auto on its own wrapper: ten numeric columns will not fit a
            phone, and the page body must never scroll sideways. */}
        <div className="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="w-[88px] font-semibold">Symbol</TableHead>
                  <TableHead className="min-w-[180px] font-semibold">Company</TableHead>
                  <TableHead className="text-right font-semibold">Match</TableHead>
                  <TableHead className="text-right font-semibold">Change %</TableHead>
                  <TableHead className="text-right font-semibold">Volume</TableHead>
                  <TableHead className="text-right font-semibold text-price-ceiling">Ceiling</TableHead>
                  <TableHead className="text-right font-semibold text-price-floor">Floor</TableHead>
                  <TableHead className="text-right font-semibold text-price-ref">Ref</TableHead>
                  <TableHead className="text-right font-semibold">High</TableHead>
                  <TableHead className="text-right font-semibold">Low</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {priceBoard.length === 0 ? (
                  <TableRow className="hover:bg-transparent">
                    <TableCell colSpan={10} className="h-32 text-center text-sm text-muted-foreground">
                      No symbols on your board yet. Add one above to start tracking it.
                    </TableCell>
                  </TableRow>
                ) : (
                  priceBoard.map((stock: PriceBoard) => {
                    const { listingInfo, matchPrice } = stock;
                    const change = calculateChangePercent(matchPrice.matchPrice, matchPrice.referencePrice);

                    // Named arguments — the previous positional call passed
                    // (price, ceiling, floor, reference) into a function that
                    // expected (price, reference, ceiling, floor).
                    const priceArgs = {
                      reference: matchPrice.referencePrice,
                      ceiling: listingInfo.ceiling,
                      floor: listingInfo.floor,
                    };
                    const matchState = getPriceState({ price: matchPrice.matchPrice, ...priceArgs });
                    const matchStyle = PRICE_STATE[matchState];

                    return (
                      <TableRow key={listingInfo.symbol} className="transition-colors hover:bg-muted/50">
                        <TableCell className={`font-semibold ${matchStyle.text}`}>{listingInfo.symbol}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {listingInfo.enOrganShortName}
                        </TableCell>

                        <PriceCell value={matchPrice.matchPrice} {...priceArgs} className="font-semibold" />

                        <TableCell className={`text-right font-mono font-semibold tabular-nums ${matchStyle.text}`}>
                          <span className="sr-only">{matchStyle.label}: </span>
                          {change.isPositive ? "+" : ""}
                          {change.percent}%
                        </TableCell>

                        <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                          {formatNumber(matchPrice.accumulatedVolume)}
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
