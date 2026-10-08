import type { PriceBoard } from "@/types/stock";

export type SectorCode = string | null;

export type SortDirection = "asc" | "desc";
export type SortState = { column: string; direction: SortDirection };

export type BoardColumn = {
  key: string;
  label: string;
  isNumeric: boolean;
  className?: string;
  value?: (stock: PriceBoard) => string | number;
};

export const EXCHANGES = ["HOSE", "HNX", "UPCOM"] as const;

// Column definitions in render order; sortable columns expose a value accessor
export const COLUMNS: BoardColumn[] = [
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
  { key: "actions", label: "", isNumeric: false, className: "w-[44px]" },
];

// Compare two board rows by the active sort column
export const compareRows = (a: PriceBoard, b: PriceBoard, sort: SortState): number => {
  const column = COLUMNS.find((candidate) => candidate.key === sort.column);
  if (!column?.value) return 0;

  const left = column.value(a);
  const right = column.value(b);
  const order =
    typeof left === "number" && typeof right === "number"
      ? left - right
      : String(left ?? "").localeCompare(String(right ?? ""), "vi");

  return sort.direction === "asc" ? order : -order;
};

// True when an axios rejection is an intentional abort
export const isRequestCanceled = (error: unknown): boolean => {
  if (!error || typeof error !== "object") return false;
  const err = error as { code?: string; name?: string };
  return err.code === "ERR_CANCELED" || err.name === "CanceledError";
};
