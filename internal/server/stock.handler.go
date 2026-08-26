package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"stockmind/internal/common"
	"stockmind/internal/database"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const vciHTTPTimeout = 10 * time.Second

type AddSymbolInPriceBoardRequest struct {
	Symbol string `json:"symbol"`
}

// FetchStockPrice is the single source of truth for calling the VCI price board:
// request shape, headers and gzip handling all live here rather than at each
// call site. It returns the untyped payload, verbatim.
func FetchStockPrice(ctx context.Context, symbols []string) (any, error) {
	respBytes, err := fetchPriceBoardJSON(ctx, symbols)
	if err != nil {
		return nil, err
	}

	var result any
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from VCI: %w", err)
	}

	return result, nil
}

// fetchPriceBoardJSON posts the symbol list to the VCI price board and returns
// the decompressed response body, so a caller can decode it into whatever shape
// it needs.
func fetchPriceBoardJSON(ctx context.Context, symbols []string) ([]byte, error) {
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

	return respBytes, nil
}

// GetLatestMatchPrice reads the price out of the VCI response at
// [0].matchPrice.matchPrice — an untyped shape, hence the assertion per level.
func GetLatestMatchPrice(ctx context.Context, ticker string) (float64, error) {
	data, err := FetchStockPrice(ctx, []string{ticker})
	if err != nil {
		return 0, fmt.Errorf("fetch price for %s: %w", ticker, err)
	}

	items, ok := data.([]any)
	if !ok || len(items) == 0 {
		return 0, fmt.Errorf("unexpected response format for %s: not an array or empty", ticker)
	}

	item, ok := items[0].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("unexpected response format for %s: first element is not an object", ticker)
	}

	matchPriceObj, ok := item["matchPrice"].(map[string]any)
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

	symbols := distinctTickers(watchlist)

	if len(symbols) == 0 {
		common.WriteJSON(w, http.StatusOK, []any{})
		return
	}

	respBytes, err := fetchPriceBoardJSON(r.Context(), symbols)
	if err != nil {
		slog.Error("[PriceBoard] price board error", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}

	var rows []map[string]any
	if err := json.Unmarshal(respBytes, &rows); err != nil {
		slog.Error("[PriceBoard] invalid price board payload", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}

	common.WriteJSON(w, http.StatusOK, dropUnresolvedRows(rows, symbols))
}

// distinctTickers lists the watchlist's tickers once each, in the order they
// first appear.
//
// The watchlist has no uniqueness constraint, so the same symbol can sit in it
// twice. VietCap answers one row per symbol asked, so a duplicate entry produces
// two identical rows that nothing downstream can tell apart - and the board
// renders rows keyed by symbol, where a repeated key lets React drop or
// duplicate rows as the sort order changes.
func distinctTickers(watchlist []database.Watchlist) []string {
	tickers := make([]string, 0, len(watchlist))
	seen := make(map[string]bool, len(watchlist))

	for _, item := range watchlist {
		if seen[item.Ticker] {
			continue
		}
		seen[item.Ticker] = true
		tickers = append(tickers, item.Ticker)
	}

	return tickers
}

// dropUnresolvedRows removes rows VietCap could not resolve to a listed symbol.
//
// VietCap answers with one row per requested symbol whether or not it knows the
// symbol; for one it does not, listingInfo is null while matchPrice is still
// populated. Such a row has no name, no ceiling and no floor, so it is unusable
// to any caller and is dropped here rather than passed on. The watchlist accepts
// free text, so a typo is all it takes to produce one.
func dropUnresolvedRows(rows []map[string]any, requested []string) []map[string]any {
	resolved := make([]map[string]any, 0, len(rows))
	seen := make(map[string]bool, len(rows))

	for _, row := range rows {
		listing, ok := row["listingInfo"].(map[string]any)
		if !ok {
			continue
		}
		if symbol, ok := listing["symbol"].(string); ok {
			seen[symbol] = true
		}
		resolved = append(resolved, row)
	}

	// Name the watchlist entries VietCap does not recognise, so a bad ticker is
	// visible in the logs instead of just missing from the board
	var unknown []string
	for _, symbol := range requested {
		if !seen[symbol] {
			unknown = append(unknown, symbol)
		}
	}
	if len(unknown) > 0 {
		slog.Warn("[PriceBoard] watchlist holds symbols VietCap does not recognise", "symbols", unknown)
	}

	return resolved
}

// POST /v1/stock/add-symbol - Add a ticker to the watchlist
//
// The watchlist table has no uniqueness constraint and the field accepts free
// text, so both checks belong here: an unlisted symbol produces a board row
// VietCap cannot resolve, and a duplicate produces two rows nothing downstream
// can tell apart.
func (s *Server) AddSymbolInPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	var req AddSymbolInPriceBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Tickers are upper case, and this arrives as whatever was typed
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Symbol is required")
		return
	}

	// Reject anything the exchanges do not list
	listed, err := s.sectorDirectory.lists(r.Context(), symbol)
	if err != nil {
		slog.Error("[PriceBoard] failed to load company directory", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to verify the symbol")
		return
	}
	if !listed {
		common.WriteJSONError(w, http.StatusBadRequest,
			fmt.Sprintf("%s is not listed on HOSE, HNX or UPCOM", symbol))
		return
	}

	// Reject a ticker the watchlist already holds
	watchlist, err := s.queries.GetWatchlist(r.Context(), 0)
	if err != nil {
		slog.Error("[PriceBoard] failed to get watchlist", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to add symbol to watchlist")
		return
	}
	if slices.Contains(distinctTickers(watchlist), symbol) {
		common.WriteJSONError(w, http.StatusConflict,
			fmt.Sprintf("%s is already on your watchlist", symbol))
		return
	}

	if _, err := s.queries.CreateWatchlistData(r.Context(), symbol); err != nil {
		slog.Error("[PriceBoard] failed to add symbol to watchlist", "symbol", symbol, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to add symbol to watchlist")
		return
	}

	common.WriteJSON(w, http.StatusCreated, "Symbol added to watchlist successfully")
}

// DELETE /v1/stock/watchlist/{id} - Remove a ticker from the watchlist
func (s *Server) DeleteSymbolFromWatchlistHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid watchlist id")
		return
	}

	removed, err := s.queries.DeleteWatchlistData(r.Context(), id)
	if err != nil {
		// A missing row is the 404 case, not a failure to delete one.
		if errors.Is(err, pgx.ErrNoRows) {
			common.WriteJSONError(w, http.StatusNotFound, "Watchlist entry not found")
			return
		}
		slog.Error("[PriceBoard] failed to remove symbol from watchlist", "id", id, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to remove symbol from watchlist")
		return
	}

	slog.Info("[PriceBoard] removed symbol from watchlist", "symbol", removed.Ticker, "id", id)
	common.WriteJSON(w, http.StatusOK, removed)
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
