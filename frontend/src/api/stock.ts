import api from ".";

export const getPriceBoard = async (symbols_list: string[]) => {
  const response = await api.post(`/stock/price-board`, { symbols: symbols_list });
  return response.data;
};
