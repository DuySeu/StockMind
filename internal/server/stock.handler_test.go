package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"stockmind/internal/database"

	"github.com/go-chi/chi/v5"
)

func TestDropUnresolvedRowsRemovesSymbolsVietCapDoesNotKnow(t *testing.T) {
	t.Parallel()

	// The watchlist accepts free text, so a typo reaches VietCap, which answers
	// with a row whose listingInfo is null while matchPrice is still populated.
	// Rendering one crashes the board on `listingInfo.ceiling`.
	rows := []map[string]any{
		{"listingInfo": map[string]any{"symbol": "FPT"}, "matchPrice": map[string]any{"matchPrice": 71000.0}},
		{"listingInfo": nil, "matchPrice": map[string]any{"matchPrice": 0.0}},
		{"listingInfo": map[string]any{"symbol": "VPB"}, "matchPrice": map[string]any{"matchPrice": 26650.0}},
	}

	got := dropUnresolvedRows(rows, []string{"FPT", "ZZZZ", "VPB"})

	if len(got) != 2 {
		t.Fatalf("len(dropUnresolvedRows(...)) = %d; want 2", len(got))
	}
	for i, row := range got {
		if row["listingInfo"] == nil {
			t.Errorf("row %d has a null listingInfo; want it dropped", i)
		}
	}
}

func TestDropUnresolvedRowsKeepsEveryResolvedRow(t *testing.T) {
	t.Parallel()

	rows := []map[string]any{
		{"listingInfo": map[string]any{"symbol": "FPT"}},
		{"listingInfo": map[string]any{"symbol": "VPB"}},
	}

	if got := dropUnresolvedRows(rows, []string{"FPT", "VPB"}); len(got) != 2 {
		t.Errorf("len = %d; want 2", len(got))
	}
}

func TestDropUnresolvedRowsHandlesAnEmptyBoard(t *testing.T) {
	t.Parallel()

	got := dropUnresolvedRows(nil, []string{"ZZZZ"})

	if got == nil {
		t.Error("dropUnresolvedRows(nil, ...) = nil; want an empty slice so it marshals as [] not null")
	}
	if len(got) != 0 {
		t.Errorf("len = %d; want 0", len(got))
	}
}

func TestDistinctTickersAsksForEachSymbolOnce(t *testing.T) {
	t.Parallel()

	// Adding the same ticker twice is not rejected anywhere, so the board has to
	// cope with it. Two rows for one symbol share a React key, which is what made
	// sorting by Symbol or Company scramble the table.
	watchlist := []database.Watchlist{
		{Ticker: "HPG"},
		{Ticker: "DGC"},
		{Ticker: "VIX"},
		{Ticker: "DGC"},
	}

	got := distinctTickers(watchlist)

	want := []string{"HPG", "DGC", "VIX"}
	if !slices.Equal(got, want) {
		t.Errorf("distinctTickers(...) = %v; want %v (first appearance wins, order kept)", got, want)
	}
}

func TestDistinctTickersHandlesAnEmptyWatchlist(t *testing.T) {
	t.Parallel()

	if got := distinctTickers(nil); len(got) != 0 {
		t.Errorf("distinctTickers(nil) = %v; want empty", got)
	}
}

// addSymbolRequest builds a POST body for the add-symbol endpoint.
func addSymbolRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/v1/stock/add-symbol", strings.NewReader(body))
}

// Both rejections below happen before the handler reaches the database, so a
// Server carrying only a warm directory is enough to drive them. A nil queries
// field is the assertion: if validation ever stops short-circuiting, these panic
// rather than quietly passing.
func TestAddSymbolRejectsATickerNoExchangeLists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "rejects an unlisted ticker", body: `{"symbol":"ZZZZ"}`, want: http.StatusBadRequest},
		{name: "rejects an empty symbol", body: `{"symbol":""}`, want: http.StatusBadRequest},
		{name: "rejects whitespace", body: `{"symbol":"   "}`, want: http.StatusBadRequest},
		{name: "rejects a malformed body", body: `not json`, want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := serverWithDirectory(entry("FPT", "HOSE", "9500", "Technology"))

			rec := httptest.NewRecorder()
			s.AddSymbolInPriceBoardHandler(rec, addSymbolRequest(tt.body))

			if rec.Code != tt.want {
				t.Errorf("AddSymbolInPriceBoardHandler(%s) status = %d; want %d, body %s",
					tt.body, rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// lists is what the add-symbol check is built on, and it is case-sensitive -
// which is why the handler upper-cases the submitted symbol first.
func TestDirectoryListsOnlyTheSymbolsItHolds(t *testing.T) {
	t.Parallel()

	directory := &symbolDirectory{
		entries: []companyEntry{
			entry("FPT", "HOSE", "9500", "Technology"),
			entry("VCB", "HOSE", "8300", "Banks"),
		},
		fetchedAt: time.Now(),
	}

	tests := []struct {
		name   string
		symbol string
		want   bool
	}{
		{name: "holds a listed symbol", symbol: "FPT", want: true},
		{name: "holds another listed symbol", symbol: "VCB", want: true},
		{name: "does not hold an unlisted symbol", symbol: "ZZZZ", want: false},
		{name: "does not hold a lower case symbol", symbol: "fpt", want: false},
		{name: "does not hold an empty symbol", symbol: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := directory.lists(t.Context(), tt.symbol)
			if err != nil {
				t.Fatalf("lists(%q) error = %v; want nil", tt.symbol, err)
			}
			if got != tt.want {
				t.Errorf("lists(%q) = %v; want %v", tt.symbol, got, tt.want)
			}
		})
	}
}

// A malformed id must be rejected before the handler reaches the database, so a
// Server with no queries is enough to drive it.
func TestDeleteSymbolFromWatchlistRejectsAMalformedID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "rejects a non-uuid id", id: "FPT"},
		{name: "rejects an empty id", id: ""},
		{name: "rejects a truncated uuid", id: "cf1aec3a-74a5-423f-876e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req := httptest.NewRequest(http.MethodDelete, "/v1/stock/watchlist/"+tt.id, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			(&Server{}).DeleteSymbolFromWatchlistHandler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("DeleteSymbolFromWatchlistHandler(%q) status = %d; want %d",
					tt.id, rec.Code, http.StatusBadRequest)
			}
		})
	}
}
