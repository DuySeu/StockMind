//go:build integration

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// The Vietnamese market has 19 ICB level-2 industries listed across HOSE, HNX
// and UPCOM. A change here means VietCap reclassified something, not a bug.
const wantListedIndustries = 19

func TestSectorsListEveryICBIndustryWithATickerCount(t *testing.T) {
	s := &Server{sectorDirectory: &symbolDirectory{}}

	rec := httptest.NewRecorder()
	s.GetSectorsHandler(rec, httptest.NewRequest(http.MethodGet, "/v1/stock/sectors", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GetSectorsHandler status = %d; want %d, body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var sectors []Sector
	if err := json.Unmarshal(rec.Body.Bytes(), &sectors); err != nil {
		t.Fatalf("decode sectors: %v", err)
	}

	if len(sectors) != wantListedIndustries {
		t.Errorf("len(sectors) = %d; want %d", len(sectors), wantListedIndustries)
	}
	for _, sector := range sectors {
		if sector.Count == 0 {
			t.Errorf("sector %s (%s) count = 0; want at least one listed ticker", sector.Code, sector.Name)
		}
		// Names now come from VietCap rather than a table in this repo.
		if sector.Name == "" {
			t.Errorf("sector %s has no name; the directory stopped carrying icbLv2.name", sector.Code)
		}
	}
}

func TestSectorPriceBoardReturnsIndustryTickersMostActiveFirst(t *testing.T) {
	s := &Server{sectorDirectory: &symbolDirectory{}}

	rec := httptest.NewRecorder()
	s.GetSectorPriceBoardHandler(rec, newSectorBoardRequest("8300", "?limit=5"))
	if rec.Code != http.StatusOK {
		t.Fatalf("GetSectorPriceBoardHandler status = %d; want %d, body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var rows []SectorBoardRow
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode board: %v", err)
	}

	if len(rows) != 5 {
		t.Fatalf("len(rows) = %d; want 5 (limit)", len(rows))
	}
	for i, row := range rows {
		if row.ListingInfo.Symbol == "" || row.MatchPrice.ReferencePrice == 0 {
			t.Errorf("row %d = %+v; want a symbol and a reference price", i, row)
		}
		// The exchange column reads listingInfo.board, which the price board
		// spells HSX where the company directory spells the same exchange HOSE.
		if !slices.Contains([]string{"HSX", "HNX", "UPCOM"}, row.ListingInfo.Board) {
			t.Errorf("row %d board = %q; want one of HSX, HNX, UPCOM", i, row.ListingInfo.Board)
		}
		if i > 0 && rows[i-1].MatchPrice.AccumulatedVolume < row.MatchPrice.AccumulatedVolume {
			t.Errorf("row %d volume %.0f > row %d volume %.0f; want descending",
				i, row.MatchPrice.AccumulatedVolume, i-1, rows[i-1].MatchPrice.AccumulatedVolume)
		}
	}
}

func TestFetchAverageVolumeReadsLiveDailyBars(t *testing.T) {
	average, err := fetchAverageVolume(t.Context(), "FPT")
	if err != nil {
		t.Fatalf("fetchAverageVolume(FPT) error = %v; want nil", err)
	}

	// FPT is among the most traded names on the exchange; anything near zero
	// means the bar series stopped arriving under the key the parser reads.
	if average < minAverageVolume {
		t.Errorf("fetchAverageVolume(FPT) = %.0f; want well over %d", average, minAverageVolume)
	}
}

// Construction & Materials is the largest industry and the one carrying the most
// dormant tickers, so it is where a broken liquidity filter shows first.
func TestSectorPriceBoardExcludesIlliquidTickers(t *testing.T) {
	s := &Server{sectorDirectory: &symbolDirectory{}}

	rec := httptest.NewRecorder()
	s.GetSectorPriceBoardHandler(rec, newSectorBoardRequest("2300", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("GetSectorPriceBoardHandler status = %d; want %d, body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var rows []SectorBoardRow
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("the filter emptied the industry; want the liquid names kept")
	}

	// Re-price the quietest rows through a cache of its own. The board is sorted
	// most active first, so a ticker that slipped past the filter is at the tail.
	tail := rows[max(0, len(rows)-15):]
	symbols := make([]string, 0, len(tail))
	for _, row := range tail {
		symbols = append(symbols, row.ListingInfo.Symbol)
	}

	averages := (&averageVolumeCache{}).averages(t.Context(), symbols)
	for _, row := range tail {
		average, priced := averages[row.ListingInfo.Symbol]
		if priced && average < minAverageVolume {
			t.Errorf("%s averages %.0f a day but is still on the board; want it dropped below %d",
				row.ListingInfo.Symbol, average, minAverageVolume)
		}
	}
}
