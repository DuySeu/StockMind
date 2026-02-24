import api from ".";

export const getPriceBoard = async (symbols_list: string[]) => {
  const response = await api.post(`/stock/price-board`, { symbols: symbols_list });
  return response.data;
};

export const getMarketResearch = async (symbols_list: string[], model: string) => {
  const response = await api.post(`/stock/research`, { tickers: symbols_list, research_model: model });
  return response.data;
};
