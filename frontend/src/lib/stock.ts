// Helper function to format numbers with commas
export const formatNumber = (num: number): string => {
  return num.toLocaleString("vi-VN");
};

// Helper function to format price (divide by 1000 for display in thousands)
export const formatPrice = (price: number): string => {
  return (price / 1000).toFixed(2);
};

// Helper function to calculate price change percentage
export const calculateChangePercent = (
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

/* ─────────────────── Vietnamese board price states ───────────────────
   The five states a HOSE/HNX board shows, in the conventional colours:
   tím = trần · xanh = tăng · vàng = tham chiếu · đỏ = giảm · xanh lam = sàn

   Deliberately a NAMED-ARGUMENT api. The previous positional signature was
   `(price, reference, ceiling?, floor?)` and every WatchList call site passed
   `(price, ceiling, floor, reference)` — so `price > reference` was really
   `price > ceiling` (never true, green never rendered) and `price < ceiling`
   was almost always true, painting nearly the whole board red. Named
   arguments make that class of mistake impossible to write.

   Colours come from the --price-* tokens in index.css, never a raw palette
   class, so they follow the dark-mode toggle. */

export type PriceState = "ceiling" | "up" | "reference" | "down" | "floor";

export const getPriceState = ({
  price,
  reference,
  ceiling,
  floor,
}: {
  price: number;
  reference: number;
  ceiling?: number;
  floor?: number;
}): PriceState => {
  if (ceiling !== undefined && price === ceiling) return "ceiling";
  if (floor !== undefined && price === floor) return "floor";
  if (price > reference) return "up";
  if (price < reference) return "down";
  return "reference";
};

/* Every state carries a `sign` and a `label` alongside its colour. Price cells
   must render one of them: ceiling/floor and reference/up sit at near-identical
   luminance, so hue is the only thing separating them and colour alone fails
   WCAG 1.4.1 for colour-blind users. */
export const PRICE_STATE: Record<
  PriceState,
  { text: string; bg: string; label: string; vi: string; sign: string }
> = {
  ceiling: { text: "text-price-ceiling", bg: "bg-price-ceiling-bg", label: "Ceiling", vi: "Trần", sign: "▲" },
  up: { text: "text-price-up", bg: "bg-price-up-bg", label: "Up", vi: "Tăng", sign: "▲" },
  reference: { text: "text-price-ref", bg: "bg-price-ref-bg", label: "Unchanged", vi: "Tham chiếu", sign: "–" },
  down: { text: "text-price-down", bg: "bg-price-down-bg", label: "Down", vi: "Giảm", sign: "▼" },
  floor: { text: "text-price-floor", bg: "bg-price-floor-bg", label: "Floor", vi: "Sàn", sign: "▼" },
};

/** Token-based text colour for a price. Replaces the old getPriceColorClass. */
export const getPriceClass = (args: {
  price: number;
  reference: number;
  ceiling?: number;
  floor?: number;
}): string => PRICE_STATE[getPriceState(args)].text;
