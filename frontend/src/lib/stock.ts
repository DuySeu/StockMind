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

// Helper function to get price color class based on comparison with reference/ceiling/floor
export const getPriceColorClass = (price: number, reference: number, ceiling?: number, floor?: number): string => {
  if (price === ceiling) return "text-purple-500";
  if (price === floor) return "text-cyan-500";
  if (price > reference) return "text-green-500";
  if (price < reference) return "text-red-500";
  return "text-yellow-500";
};
