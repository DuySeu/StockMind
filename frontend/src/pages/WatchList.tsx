import { addSymbolInPriceBoard, getPriceBoard } from "@/api/stock";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { PriceBoard } from "@/types/stock";
import { Plus, RotateCw } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";

// Helper function to format numbers with commas
const formatNumber = (num: number): string => {
  return num.toLocaleString("vi-VN");
};

// Helper function to format price (divide by 1000 for display in thousands)
const formatPrice = (price: number): string => {
  return (price / 1000).toFixed(2);
};

// Helper function to calculate price change percentage
const calculateChangePercent = (
  currentPrice: number,
  referencePrice: number,
): { percent: string; isPositive: boolean; isNeutral: boolean } => {
  const change = ((currentPrice - referencePrice) / referencePrice) * 100;
  return {
    percent: change.toFixed(2),
    isPositive: change > 0,
    isNeutral: change === 0,
  };
};

// Helper function to get price color class based on comparison with reference/ceiling/floor
const getPriceColorClass = (price: number, ceiling: number, floor: number, reference: number): string => {
  if (price === ceiling) return "text-purple-500";
  if (price === floor) return "text-cyan-500";
  if (price > reference) return "text-green-500";
  if (price < reference) return "text-red-500";
  return "text-yellow-500";
};

const WatchListPage = () => {
  const [priceBoard, setPriceBoard] = useState<PriceBoard[]>([]);
  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });

  useEffect(() => {
    getPriceBoard().then((res) => {
      setPriceBoard(res);
    });
  }, []);

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
    <div className="p-3 space-y-4">
      <h1 className="text-2xl font-bold text-primary">Watchlist</h1>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2">
          <FormField
            control={form.control}
            name="symbols"
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input {...field} className="text-secondary-foreground" placeholder="Enter symbols to watch..." />
                </FormControl>
              </FormItem>
            )}
          />
          <Button type="submit" size="icon">
            <Plus />
          </Button>
          <Button type="button" size="icon" onClick={handleRefresh}>
            <RotateCw />
          </Button>
        </form>
      </Form>
      <div className="rounded-lg border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/50">
              <TableHead className="w-[80px] font-bold">Symbol</TableHead>
              <TableHead className="min-w-[180px] font-bold">Company</TableHead>
              <TableHead className="text-right font-bold">Match</TableHead>
              <TableHead className="text-right font-bold">Change %</TableHead>
              <TableHead className="text-right font-bold">Volume</TableHead>
              <TableHead className="text-right font-bold text-purple-500">Ceiling</TableHead>
              <TableHead className="text-right font-bold text-cyan-500">Floor</TableHead>
              <TableHead className="text-right font-bold text-yellow-500">Ref</TableHead>
              <TableHead className="text-right font-bold">High</TableHead>
              <TableHead className="text-right font-bold">Low</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody className="text-secondary-foreground">
            {priceBoard.map((stock: PriceBoard) => {
              const { listingInfo, matchPrice } = stock;
              const change = calculateChangePercent(matchPrice.matchPrice, matchPrice.referencePrice);
              const priceColor = getPriceColorClass(
                matchPrice.matchPrice,
                listingInfo.ceiling,
                listingInfo.floor,
                matchPrice.referencePrice,
              );

              return (
                <TableRow key={listingInfo.symbol} className="hover:bg-muted/30 transition-colors">
                  <TableCell className={`font-bold ${priceColor}`}>{listingInfo.symbol}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{listingInfo.enOrganShortName}</TableCell>
                  <TableCell className={`text-right font-semibold ${priceColor}`}>
                    {formatPrice(matchPrice.matchPrice)}
                  </TableCell>
                  <TableCell
                    className={`text-right font-semibold ${
                      change.isNeutral ? "text-yellow-500" : change.isPositive ? "text-green-500" : "text-red-500"
                    }`}
                  >
                    {change.isPositive ? "+" : ""}
                    {change.percent}%
                  </TableCell>
                  <TableCell className="text-right">{formatNumber(matchPrice.accumulatedVolume)}</TableCell>
                  <TableCell className="text-right text-purple-500">{formatPrice(listingInfo.ceiling)}</TableCell>
                  <TableCell className="text-right text-cyan-500">{formatPrice(listingInfo.floor)}</TableCell>
                  <TableCell className="text-right text-yellow-500">{formatPrice(matchPrice.referencePrice)}</TableCell>
                  <TableCell
                    className={`text-right ${getPriceColorClass(matchPrice.highest, listingInfo.ceiling, listingInfo.floor, matchPrice.referencePrice)}`}
                  >
                    {formatPrice(matchPrice.highest)}
                  </TableCell>
                  <TableCell
                    className={`text-right ${getPriceColorClass(matchPrice.lowest, listingInfo.ceiling, listingInfo.floor, matchPrice.referencePrice)}`}
                  >
                    {formatPrice(matchPrice.lowest)}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};

export default WatchListPage;
