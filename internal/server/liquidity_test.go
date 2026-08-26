package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// boardRow builds one price board row for the given symbol.
func boardRow(symbol string) SectorBoardRow {
	return SectorBoardRow{ListingInfo: SectorListingInfo{Symbol: symbol}}
}

// dailyBars lays the given volumes on consecutive days ending yesterday, which
// is the shape a normally trading ticker has.
func dailyBars(volumes ...float64) []bar {
	bars := make([]bar, 0, len(volumes))
	for i, volume := range volumes {
		day := time.Now().AddDate(0, 0, -(len(volumes) - i))
		bars = append(bars, bar{at: day, volume: volume})
	}
	return bars
}

// bar is one daily candle: when it traded and how much.
type bar struct {
	at     time.Time
	volume float64
}

// withChartServer points the daily-bar endpoint at a stub returning the given
// bars per symbol, and counts how many requests reach it. A symbol absent from
// the map answers with an empty array, which is how VietCap reports a ticker it
// has no history for.
func withChartServer(t *testing.T, bars map[string][]bar) *atomic.Int64 {
	t.Helper()

	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		var req struct {
			Symbols []string `json:"symbols"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Symbols) != 1 {
			http.Error(w, "one symbol per request", http.StatusBadRequest)
			return
		}

		series := []map[string]any{}
		if candles, ok := bars[req.Symbols[0]]; ok {
			times := make([]string, 0, len(candles))
			volumes := make([]float64, 0, len(candles))
			for _, candle := range candles {
				times = append(times, strconv.FormatInt(candle.at.Unix(), 10))
				volumes = append(volumes, candle.volume)
			}
			series = append(series, map[string]any{"t": times, "v": volumes})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(series)
	}))
	t.Cleanup(srv.Close)

	// chartURL is package state, so a test that swaps it cannot be parallel
	original := chartURL
	chartURL = srv.URL
	t.Cleanup(func() { chartURL = original })

	return &calls
}

func TestDropIlliquidRowsRemovesTickersUnderTheMinimumAverage(t *testing.T) {
	t.Parallel()

	rows := []SectorBoardRow{boardRow("HPG"), boardRow("BCC"), boardRow("VCG"), boardRow("NBB")}
	averages := map[string]float64{
		"HPG": 21_979_005,
		"BCC": 29_455,
		"VCG": 2_352_380,
		"NBB": 22_710,
	}

	got := dropIlliquidRows(rows, averages)

	if len(got) != 2 {
		t.Fatalf("len(dropIlliquidRows(...)) = %d; want 2", len(got))
	}
	if got[0].ListingInfo.Symbol != "HPG" || got[1].ListingInfo.Symbol != "VCG" {
		t.Errorf("kept %q and %q; want HPG and VCG in order",
			got[0].ListingInfo.Symbol, got[1].ListingInfo.Symbol)
	}
}

func TestDropIlliquidRowsKeepsATickerExactlyAtTheMinimum(t *testing.T) {
	t.Parallel()

	// The threshold is a floor, not a bar to clear: "below 100k" excludes 100k.
	got := dropIlliquidRows(
		[]SectorBoardRow{boardRow("EDGE"), boardRow("UNDER")},
		map[string]float64{"EDGE": minAverageVolume, "UNDER": minAverageVolume - 1},
	)

	if len(got) != 1 || got[0].ListingInfo.Symbol != "EDGE" {
		t.Errorf("dropIlliquidRows kept %+v; want only EDGE", got)
	}
}

// A ticker whose history could not be fetched must stay on the board. Treating
// an absent average as zero would hide a liquid stock every time one request
// times out.
func TestDropIlliquidRowsKeepsATickerWithNoAverage(t *testing.T) {
	t.Parallel()

	got := dropIlliquidRows([]SectorBoardRow{boardRow("FPT")}, map[string]float64{})

	if len(got) != 1 {
		t.Fatalf("len = %d; want 1 - an unpriced ticker was dropped as illiquid", len(got))
	}
}

func TestDropIlliquidRowsMarshalsAnAllIlliquidSectorAsAnEmptyList(t *testing.T) {
	t.Parallel()

	got := dropIlliquidRows([]SectorBoardRow{boardRow("BCC")}, map[string]float64{"BCC": 1_000})

	if got == nil {
		t.Fatal("dropIlliquidRows() = nil; want an empty slice so it marshals as [] not null")
	}
	if len(got) != 0 {
		t.Errorf("len = %d; want 0", len(got))
	}
}

func TestAveragesMeansTheDailyVolumeSeries(t *testing.T) {
	// Four consecutive sessions totalling 1000, spread across the window's weekdays
	withChartServer(t, map[string][]bar{"FPT": dailyBars(100, 200, 300, 400)})

	sessions := float64(tradingSessionsSince(time.Now().Add(-averageVolumeWindow)))
	want := 1000 / sessions

	cache := &averageVolumeCache{}
	got := cache.averages(t.Context(), []string{"FPT"})

	if got["FPT"] != want {
		t.Errorf("averages()[FPT] = %v; want %v (1000 over %v sessions)", got["FPT"], want, sessions)
	}
}

// The bug this window exists for: VietCap answers with the last bars that exist,
// however old. A ticker suspended months ago used to average like a liquid stock
// and stayed on the board showing a volume of zero.
func TestAveragesTreatsATickerDormantSinceBeforeTheWindowAsIlliquid(t *testing.T) {
	longAgo := time.Now().AddDate(0, 0, -150)
	dormant := []bar{
		{at: longAgo, volume: 5_000_000},
		{at: longAgo.AddDate(0, 0, 1), volume: 4_000_000},
		{at: longAgo.AddDate(0, 0, 2), volume: 6_000_000},
	}
	withChartServer(t, map[string][]bar{"PSH": dormant})

	cache := &averageVolumeCache{}
	got := cache.averages(t.Context(), []string{"PSH"})

	if got["PSH"] != 0 {
		t.Errorf("averages()[PSH] = %v; want 0 - bars from outside the window were counted", got["PSH"])
	}
	if kept := dropIlliquidRows([]SectorBoardRow{boardRow("PSH")}, got); len(kept) != 0 {
		t.Error("a ticker with no volume inside the window is still on the board")
	}
}

// A ticker the exchange restricts to one session a week trades on paper but is
// not something anyone can get in and out of. Averaging over its traded sessions
// would report it as liquid; averaging over the window's sessions does not.
func TestAveragesDividesByTheWindowsSessionsNotTheOnesTraded(t *testing.T) {
	weekly := make([]bar, 0, 4)
	for week := 1; week <= 4; week++ {
		weekly = append(weekly, bar{at: time.Now().AddDate(0, 0, -7*week), volume: 400_000})
	}
	withChartServer(t, map[string][]bar{"BGE": weekly})

	cache := &averageVolumeCache{}
	got := cache.averages(t.Context(), []string{"BGE"})

	if got["BGE"] >= minAverageVolume {
		t.Errorf("averages()[BGE] = %v; want under %d - it was divided by its four traded sessions",
			got["BGE"], minAverageVolume)
	}
}

// An unpriced ticker must be absent from the result, never present as a zero -
// a zero average is indistinguishable from a genuinely dead stock and would
// take the ticker off the board.
func TestAveragesOmitsATickerWithNoHistoryRatherThanReportingZero(t *testing.T) {
	withChartServer(t, map[string][]bar{"FPT": dailyBars(1_000_000)})

	cache := &averageVolumeCache{}
	got := cache.averages(t.Context(), []string{"FPT", "NOHIST"})

	if _, present := got["NOHIST"]; present {
		t.Errorf("averages() reported NOHIST as %v; want it absent", got["NOHIST"])
	}
	if got["FPT"] == 0 {
		t.Error("averages()[FPT] = 0; want its volume - one failed symbol lost the whole batch")
	}
}

func TestAveragesServesARepeatedTickerFromTheCache(t *testing.T) {
	calls := withChartServer(t, map[string][]bar{
		"FPT": dailyBars(500_000), "VCB": dailyBars(600_000),
	})

	cache := &averageVolumeCache{}
	first := cache.averages(t.Context(), []string{"FPT", "VCB"})
	got := cache.averages(t.Context(), []string{"FPT", "VCB"})

	if calls.Load() != 2 {
		t.Errorf("chart endpoint called %d times; want 2 - the second lookup refetched", calls.Load())
	}
	if got["FPT"] != first["FPT"] || got["VCB"] != first["VCB"] {
		t.Errorf("cached averages = %v; want the first call's %v", got, first)
	}
}

// Only the tickers a sector adds should cost a request. Switching between two
// industries that overlap must not re-price the names they share.
func TestAveragesFetchesOnlyTheTickersItHasNotSeen(t *testing.T) {
	calls := withChartServer(t, map[string][]bar{
		"FPT": dailyBars(500_000), "VCB": dailyBars(600_000), "SSI": dailyBars(700_000),
	})

	cache := &averageVolumeCache{}
	cache.averages(t.Context(), []string{"FPT", "VCB"})
	got := cache.averages(t.Context(), []string{"VCB", "SSI"})

	if calls.Load() != 3 {
		t.Errorf("chart endpoint called %d times; want 3 - VCB was priced twice", calls.Load())
	}
	if got["VCB"] == 0 || got["SSI"] == 0 {
		t.Errorf("averages = %v; want both VCB and SSI priced", got)
	}
}

func TestAveragesRefetchesOnceTheEntryHasExpired(t *testing.T) {
	calls := withChartServer(t, map[string][]bar{"FPT": dailyBars(500_000)})

	cache := &averageVolumeCache{byTicker: map[string]tickerVolume{
		"FPT": {average: 1, fetchedAt: time.Now().Add(-averageVolumeTTL - time.Minute)},
	}}

	got := cache.averages(t.Context(), []string{"FPT"})

	if calls.Load() != 1 {
		t.Errorf("chart endpoint called %d times; want 1 - a stale entry was served", calls.Load())
	}
	if got["FPT"] == 1 {
		t.Error("averages()[FPT] still holds the stale value; want the refetched one")
	}
}
