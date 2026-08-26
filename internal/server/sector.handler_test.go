package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// entry builds one directory row for the given symbol and industry.
func entry(symbol, floor, industryCode, industryName string) companyEntry {
	row := companyEntry{Symbol: symbol, Floor: floor}
	row.Industry.Code = industryCode
	row.Industry.Name = industryName
	return row
}

// serverWithDirectory returns a Server whose sector cache is already warm, so
// the handlers under test make no network call of their own.
func serverWithDirectory(entries ...companyEntry) *Server {
	return &Server{sectorDirectory: &symbolDirectory{
		entries:   entries,
		fetchedAt: time.Now(),
	}}
}

// newSectorBoardRequest builds a price-board request carrying the sector code
// chi would have parsed out of the route.
func newSectorBoardRequest(code, query string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)

	req := httptest.NewRequest(http.MethodGet, "/v1/stock/sectors/"+code+"/price-board"+query, nil)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestSectorPriceBoardRejectsACodeNoCompanyBelongsTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
	}{
		{name: "rejects a code outside the directory", code: "9999"},
		{name: "rejects an empty code", code: ""},
		{name: "rejects a non-numeric code", code: "banking"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := serverWithDirectory(entry("FPT", "HOSE", "9500", "Technology"))

			rec := httptest.NewRecorder()
			s.GetSectorPriceBoardHandler(rec, newSectorBoardRequest(tt.code, ""))

			if rec.Code != http.StatusBadRequest {
				t.Errorf("GetSectorPriceBoardHandler(%q) status = %d; want %d", tt.code, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestSectorsHandlerGroupsTheDirectoryIntoIndustries(t *testing.T) {
	t.Parallel()

	s := serverWithDirectory(
		entry("FPT", "HOSE", "9500", "Technology"),
		entry("CMG", "HOSE", "9500", "Technology"),
		entry("ELC", "HNX", "9500", "Technology"),
		entry("VCB", "HOSE", "8300", "Banks"),
		entry("TCB", "HOSE", "8300", "Banks"),
		entry("PLX", "HOSE", "0500", "Oil & Gas"),
	)

	rec := httptest.NewRecorder()
	s.GetSectorsHandler(rec, httptest.NewRequest(http.MethodGet, "/v1/stock/sectors", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	var sectors []Sector
	if err := json.Unmarshal(rec.Body.Bytes(), &sectors); err != nil {
		t.Fatalf("decode sectors: %v", err)
	}

	// Largest industry first, and names come from the directory, not a local table
	want := []Sector{
		{Code: "9500", Name: "Technology", Count: 3},
		{Code: "8300", Name: "Banks", Count: 2},
		{Code: "0500", Name: "Oil & Gas", Count: 1},
	}
	if len(sectors) != len(want) {
		t.Fatalf("len(sectors) = %d; want %d", len(sectors), len(want))
	}
	for i, sector := range sectors {
		if sector != want[i] {
			t.Errorf("sectors[%d] = %+v; want %+v", i, sector, want[i])
		}
	}
}

func TestSectorsFromOrdersEquallySizedIndustriesByCode(t *testing.T) {
	t.Parallel()

	// Ties must break deterministically or the filter row reshuffles between calls.
	entries := []companyEntry{
		entry("VCB", "HOSE", "8300", "Banks"),
		entry("PLX", "HOSE", "0500", "Oil & Gas"),
		entry("SSI", "HOSE", "8700", "Financial Services"),
	}

	first := sectorsFrom(entries)
	second := sectorsFrom(entries)

	wantCodes := []string{"0500", "8300", "8700"}
	for i, code := range wantCodes {
		if first[i].Code != code {
			t.Errorf("first[%d].Code = %q; want %q", i, first[i].Code, code)
		}
		if second[i] != first[i] {
			t.Errorf("ordering is not stable at %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}

func TestDropUnquotedRowsRemovesTickersThePriceBoardNoLongerQuotes(t *testing.T) {
	t.Parallel()

	// VietCap's company directory outlives its price board, so an industry can
	// name tickers that come back as an all-zero row with no symbol. Rendering
	// one produces a blank table line showing 0.00 prices.
	rows := []SectorBoardRow{
		{ListingInfo: SectorListingInfo{Symbol: "VCG"}, MatchPrice: SectorMatchPrice{AccumulatedVolume: 100}},
		{},
		{ListingInfo: SectorListingInfo{Symbol: "CII"}, MatchPrice: SectorMatchPrice{AccumulatedVolume: 300}},
		{},
	}

	got := dropUnquotedRows(rows)

	if len(got) != 2 {
		t.Fatalf("len(dropUnquotedRows(...)) = %d; want 2", len(got))
	}
	if got[0].ListingInfo.Symbol != "VCG" || got[1].ListingInfo.Symbol != "CII" {
		t.Errorf("kept %q and %q; want VCG and CII in order", got[0].ListingInfo.Symbol, got[1].ListingInfo.Symbol)
	}
}

func TestDropUnquotedRowsMarshalsAnAllUnquotedSectorAsAnEmptyList(t *testing.T) {
	t.Parallel()

	got := dropUnquotedRows([]SectorBoardRow{{}, {}})

	if got == nil {
		t.Fatal("dropUnquotedRows() = nil; want an empty slice so it marshals as [] not null")
	}
	if len(got) != 0 {
		t.Errorf("len = %d; want 0", len(got))
	}
}
