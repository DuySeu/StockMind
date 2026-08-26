import api from ".";

export const getPriceBoard = async (limit?: number) => {
  const response = await api.get(`/stock/price-board${limit ? `?limit=${limit}` : ""}`);
  return response.data;
};

export const getWatchlist = async () => {
  const response = await api.get(`/stock/watchlist`);
  return response.data;
};

// Fetch the ICB industries the price board can be filtered by
export const getSectors = async () => {
  const response = await api.get(`/stock/sectors`);
  return response.data;
};

// Fetch the price board for one ICB industry, most actively traded first
export const getSectorPriceBoard = async (code: string, limit?: number) => {
  const response = await api.get(`/stock/sectors/${code}/price-board${limit ? `?limit=${limit}` : ""}`);
  return response.data;
};

// Remove one watchlist entry by its row id
export const deleteWatchlistSymbol = async (id: string) => {
  const response = await api.delete(`/stock/watchlist/${id}`);
  return response.data;
};

export const addSymbolInPriceBoard = async (symbol: string) => {
  const response = await api.post(`/stock/add-symbol`, { symbol });
  return response.data;
};

export const getMarketResearch = async (symbols_list: string[], model: string) => {
  const response = await api.post(`/stock/research`, { tickers: symbols_list, research_model: model });
  return response.data;
};

export const getResearchReport = async () => {
  const response = await api.get(`/stock/research-reports`);
  return response.data;
};

export const getResearchReportById = async (report_id: string) => {
  const response = await api.get(`/stock/research-reports/${report_id}`);
  return response.data;
};

export interface ResearchProgressEvent {
  ticker: string;
  step: string;
  message: string;
  progress: number;
}

export interface StreamMarketResearchCallbacks {
  onProgress: (event: ResearchProgressEvent) => void;
  onComplete: (data: any) => void;
  onError: (error: string) => void;
}

export const streamMarketResearch = (
  symbolsList: string[],
  model: string,
  callbacks: StreamMarketResearchCallbacks,
): AbortController => {
  const controller = new AbortController();
  let parsedLength = 0;

  const parseSSEChunk = (rawText: string) => {
    // Only process the new portion of the cumulative response
    const newText = rawText.slice(parsedLength);
    parsedLength = rawText.length;

    const lines = newText.split("\n");
    for (const line of lines) {
      if (!line.startsWith("data: ")) continue;
      const jsonStr = line.slice(6).trim();
      if (!jsonStr) continue;

      try {
        const event = JSON.parse(jsonStr);
        switch (event.type) {
          case "progress":
            callbacks.onProgress(event.data);
            break;
          case "result":
            callbacks.onComplete(event.data);
            break;
          case "error":
            callbacks.onError(event.data?.message ?? "Unknown error");
            break;
        }
      } catch {
        // ignore malformed lines
      }
    }
  };

  api
    .post(
      "/stock/research/stream",
      { tickers: symbolsList, research_model: model },
      {
        signal: controller.signal,
        responseType: "text",
        onDownloadProgress: (progressEvent) => {
          const rawText = (progressEvent.event?.target as XMLHttpRequest)?.responseText;
          if (rawText) {
            parseSSEChunk(rawText);
          }
        },
      },
    )
    .catch((err) => {
      if (err.name !== "CanceledError" && err.code !== "ERR_CANCELED") {
        callbacks.onError(err.message ?? "Connection failed");
      }
    });

  return controller;
};
