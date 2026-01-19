export type PriceBoard = {
  listingInfo: ListingInfo;
  matchPrice: MatchPrice;
};

type ListingInfo = {
  ceiling: number;
  floor: number;
  symbol: string;
  enOrganShortName: string;
};

type MatchPrice = {
  matchPrice: number;
  referencePrice: number;
  accumulatedVolume: number;
  highest: number;
  lowest: number;
};
