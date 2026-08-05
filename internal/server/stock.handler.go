package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"stockmind/internal/common"
	"stockmind/internal/database"
	"strconv"
	"time"
)

const vciHTTPTimeout = 10 * time.Second

type AddSymbolInPriceBoardRequest struct {
	Symbol string `json:"symbol"`
}

// FetchStockPrice is the single source of truth for calling the VCI price board:
// request shape, headers and gzip handling all live here rather than at each
// call site.
func FetchStockPrice(ctx context.Context, symbols []string) (interface{}, error) {
	body, err := json.Marshal(struct {
		Symbols []string `json:"symbols"`
	}{Symbols: symbols})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal symbols: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s%s", common.TRADING_URL, common.PRICE_BOARD_URL),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range common.VCI_HEADERS {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: vciHTTPTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stock prices: %w", err)
	}
	defer resp.Body.Close()

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from VCI: %w", err)
	}

	return result, nil
}

// GetLatestMatchPrice reads the price out of the VCI response at
// [0].matchPrice.matchPrice — an untyped shape, hence the assertion per level.
func GetLatestMatchPrice(ctx context.Context, ticker string) (float64, error) {
	data, err := FetchStockPrice(ctx, []string{ticker})
	if err != nil {
		return 0, fmt.Errorf("fetch price for %s: %w", ticker, err)
	}

	items, ok := data.([]interface{})
	if !ok || len(items) == 0 {
		return 0, fmt.Errorf("unexpected response format for %s: not an array or empty", ticker)
	}

	item, ok := items[0].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected response format for %s: first element is not an object", ticker)
	}

	matchPriceObj, ok := item["matchPrice"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("missing matchPrice object for %s", ticker)
	}

	price, ok := matchPriceObj["matchPrice"].(float64)
	if !ok {
		return 0, fmt.Errorf("matchPrice is not a number for %s", ticker)
	}

	return price, nil
}

// GET /v1/stock/price-board - Get live prices for every ticker on the watchlist
func (s *Server) GetPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")

	limit := 0
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var watchlist []database.Watchlist
	var err error

	watchlist, err = s.queries.GetWatchlist(r.Context(), int32(limit))

	if err != nil {
		slog.Error("[PriceBoard] failed to get watchlist", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get watchlist")
		return
	}

	symbols := make([]string, 0, len(watchlist))
	for _, item := range watchlist {
		symbols = append(symbols, item.Ticker)
	}

	if len(symbols) == 0 {
		common.WriteJSON(w, http.StatusOK, []interface{}{})
		return
	}

	priceBoard, err := FetchStockPrice(r.Context(), symbols)
	if err != nil {
		slog.Error("[PriceBoard] price board error", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}

	common.WriteJSON(w, http.StatusOK, priceBoard)
}

// POST /v1/stock/add-symbol - Add a ticker to the watchlist
func (s *Server) AddSymbolInPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	var req AddSymbolInPriceBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err := s.queries.CreateWatchlistData(r.Context(), req.Symbol)
	if err != nil {
		slog.Error("[PriceBoard] failed to add symbol to watchlist", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to add symbol to watchlist")
		return
	}

	common.WriteJSON(w, http.StatusCreated, "Symbol added to watchlist successfully")
}

// GET /v1/stock/watchlist - Get the watchlist
func (s *Server) GetWatchlistHandler(w http.ResponseWriter, r *http.Request) {
	watchlist, err := s.queries.GetWatchlist(r.Context(), 0)
	if err != nil {
		slog.Error("[PriceBoard] failed to get watchlist", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get watchlist")
		return
	}

	common.WriteJSON(w, http.StatusOK, watchlist)
}
