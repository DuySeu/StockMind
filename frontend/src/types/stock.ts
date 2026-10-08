export type Sector = {
  code: string;
  name: string;
  count: number;
};

export type WatchlistEntry = {
  id: string;
  ticker: string;
  created_at: string;
  position: number;
};

/* One benchmark's current standing. Quoted in index points, not VND, so it is
   never formatted through formatPrice. */
export type MarketIndex = {
  symbol: string;
  value: number;
  reference: number;
  change: number;
  changePercent: number;
  volume: number;
  high: number;
  low: number;
};

export type PriceBoard = {
  listingInfo: ListingInfo;
  matchPrice: MatchPrice;
};

type ListingInfo = {
  ceiling: number;
  floor: number;
  symbol: string;
  enOrganShortName: string;
  board: string;
};

type MatchPrice = {
  matchPrice: number;
  referencePrice: number;
  accumulatedVolume: number;
  highest: number;
  lowest: number;
};
