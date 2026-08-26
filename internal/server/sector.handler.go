package server

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"stockmind/internal/common"

	"github.com/go-chi/chi/v5"
)

// The company directory only changes on a listing day, so one fetch per half-day
// per process is plenty for a ~1MB payload.
const symbolDirectoryTTL = 12 * time.Hour

// errNoTradableSymbols reports a directory that parsed but held nothing usable,
// which means VietCap changed its shape rather than that the market is empty.
var errNoTradableSymbols = errors.New("company directory returned no tradable symbols")

// tradableFloors are the exchanges whose symbols have a live price board. The
// rest of the directory is OTC, delisted and suspended rows.
//
// Note the vocabulary differs across VietCap's own services: this directory says
// HOSE where the price board's listingInfo.board says HSX.
var tradableFloors = []string{"HOSE", "HNX", "UPCOM"}

// Sector is one ICB level-2 industry the price board can be filtered by.
type Sector struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// companyEntry is the slice of VietCap's company directory the sector filter
// reads. Industry names arrive with the data, so nothing here is hardcoded.
type companyEntry struct {
	Symbol   string `json:"code"`
	Floor    string `json:"floor"`
	IsIndex  bool   `json:"isIndex"`
	Industry struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"icbLv2"`
}

// symbolDirectory caches the tradable stocks of VietCap's company directory
// behind a TTL. The mutex doubles as a single-flight gate: a cold cache under
// concurrent requests refetches once, not once per request.
type symbolDirectory struct {
	mu        sync.Mutex
	entries   []companyEntry
	fetchedAt time.Time
}

// SectorBoardRow is one row of the sector price board. VCI returns ~45 fields
// per row and the table reads nine of them, so the response is projected down
// to the shape the frontend already declares for the watchlist board.
type SectorBoardRow struct {
	ListingInfo SectorListingInfo `json:"listingInfo"`
	MatchPrice  SectorMatchPrice  `json:"matchPrice"`
}

type SectorListingInfo struct {
	Symbol           string  `json:"symbol"`
	EnOrganShortName string  `json:"enOrganShortName"`
	Board            string  `json:"board"`
	Ceiling          float64 `json:"ceiling"`
	Floor            float64 `json:"floor"`
}

type SectorMatchPrice struct {
	MatchPrice        float64 `json:"matchPrice"`
	ReferencePrice    float64 `json:"referencePrice"`
	AccumulatedVolume float64 `json:"accumulatedVolume"`
	Highest           float64 `json:"highest"`
	Lowest            float64 `json:"lowest"`
}

// GET /v1/stock/sectors - List the ICB industries the price board can be filtered by
func (s *Server) GetSectorsHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := s.sectorDirectory.load(r.Context())
	if err != nil {
		slog.Error("[Sector] failed to load company directory", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to load sectors")
		return
	}

	common.WriteJSON(w, http.StatusOK, sectorsFrom(entries))
}

// GET /v1/stock/sectors/{code}/price-board - Get live prices for one industry's tickers
func (s *Server) GetSectorPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "Sector code is required")
		return
	}

	limit := 0
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed > 0 {
		limit = parsed
	}

	entries, err := s.sectorDirectory.load(r.Context())
	if err != nil {
		slog.Error("[Sector] failed to load company directory", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to load sectors")
		return
	}

	// Collect the industry's tickers
	symbols := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Industry.Code == code {
			symbols = append(symbols, entry.Symbol)
		}
	}
	if len(symbols) == 0 {
		common.WriteJSONError(w, http.StatusBadRequest, "Unknown sector code")
		return
	}

	respBytes, err := fetchPriceBoardJSON(r.Context(), symbols)
	if err != nil {
		slog.Error("[Sector] price board error", "sector", code, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}

	var rows []SectorBoardRow
	if err := json.Unmarshal(respBytes, &rows); err != nil {
		slog.Error("[Sector] invalid price board payload", "sector", code, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}

	quoted := dropUnquotedRows(rows)
	if dropped := len(rows) - len(quoted); dropped > 0 {
		slog.Info("[Sector] dropped tickers the price board no longer quotes",
			"sector", code, "dropped", dropped, "kept", len(quoted))
	}

	// Drop the tickers too thin to trade, by their average volume rather than
	// today's: an hour into the session every accumulated volume is still small
	quotedSymbols := make([]string, 0, len(quoted))
	for _, row := range quoted {
		quotedSymbols = append(quotedSymbols, row.ListingInfo.Symbol)
	}

	rows = dropIlliquidRows(quoted, s.averageVolume.averages(r.Context(), quotedSymbols))
	if dropped := len(quoted) - len(rows); dropped > 0 {
		slog.Info("[Sector] dropped illiquid tickers",
			"sector", code, "dropped", dropped, "kept", len(rows), "minAverageVolume", minAverageVolume)
	}

	// Most active first, so an industry's liquid names lead the table
	slices.SortFunc(rows, func(a, b SectorBoardRow) int {
		return cmp.Compare(b.MatchPrice.AccumulatedVolume, a.MatchPrice.AccumulatedVolume)
	})

	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}

	common.WriteJSON(w, http.StatusOK, rows)
}

// dropUnquotedRows removes rows the price board could not resolve to a listed
// symbol.
//
// The company directory outlives the price board: it still names tickers
// VietCap no longer quotes, and those come back as an all-zero row with an empty
// symbol. Rendering one is a blank table line showing 0.00 prices.
func dropUnquotedRows(rows []SectorBoardRow) []SectorBoardRow {
	quoted := make([]SectorBoardRow, 0, len(rows))
	for _, row := range rows {
		if row.ListingInfo.Symbol == "" {
			continue
		}
		quoted = append(quoted, row)
	}
	return quoted
}

// sectorsFrom groups the directory into its ICB level-2 industries, largest first.
func sectorsFrom(entries []companyEntry) []Sector {
	counts := make(map[string]int)
	names := make(map[string]string)
	for _, entry := range entries {
		counts[entry.Industry.Code]++
		names[entry.Industry.Code] = entry.Industry.Name
	}

	sectors := make([]Sector, 0, len(counts))
	for code, count := range counts {
		sectors = append(sectors, Sector{Code: code, Name: names[code], Count: count})
	}

	// Biggest industries first, then by code so the order is stable across calls
	slices.SortFunc(sectors, func(a, b Sector) int {
		if a.Count != b.Count {
			return cmp.Compare(b.Count, a.Count)
		}
		return cmp.Compare(a.Code, b.Code)
	})

	return sectors
}

// lists reports whether the company directory holds the given symbol, which is
// the test for "is this a real ticker somebody can put on a price board". The
// directory is already cached for the sector filter, so this costs nothing.
func (d *symbolDirectory) lists(ctx context.Context, symbol string) (bool, error) {
	entries, err := d.load(ctx)
	if err != nil {
		return false, err
	}

	return slices.ContainsFunc(entries, func(entry companyEntry) bool {
		return entry.Symbol == symbol
	}), nil
}

// load returns the cached company directory, refetching it once the TTL expires.
func (d *symbolDirectory) load(ctx context.Context) ([]companyEntry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.entries != nil && time.Since(d.fetchedAt) < symbolDirectoryTTL {
		return d.entries, nil
	}

	all, err := common.FetchIQInsight[[]companyEntry](ctx, common.SEARCH_BAR_URL)
	if err != nil {
		return nil, err
	}

	// Keep only listed companies carrying an industry
	tradable := make([]companyEntry, 0, len(all))
	for _, entry := range all {
		if entry.IsIndex || entry.Industry.Code == "" || !slices.Contains(tradableFloors, entry.Floor) {
			continue
		}
		tradable = append(tradable, entry)
	}

	if len(tradable) == 0 {
		return nil, errNoTradableSymbols
	}

	d.entries = tradable
	d.fetchedAt = time.Now()

	return tradable, nil
}
