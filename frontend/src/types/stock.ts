export type Sector = {
  code: string;
  name: string;
  count: number;
};

export type WatchlistEntry = {
  id: string;
  ticker: string;
  created_at: string;
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
