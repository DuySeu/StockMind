package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"stockmind/internal/common"
)

// marketIndices are the three benchmarks the watchlist header shows, in the
// order they are rendered.
var marketIndices = []string{"VNINDEX", "VN30", "VN100"}

// indexBars is how many daily bars an index quote needs: the current session and
// the one before it, which is the reference the change is measured against.
const indexBars = 2

// MarketIndex is one benchmark's current standing, in index points rather than
// VND - a card renders it as-is, with no division by a thousand.
type MarketIndex struct {
	Symbol        string  `json:"symbol"`
	Value         float64 `json:"value"`
	Reference     float64 `json:"reference"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Volume        float64 `json:"volume"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
}

// GET /v1/stock/indices - Get the current standing of VNINDEX, VN30 and VN100
//
// The price board answers for an index with every field zeroed, so these come
// from the daily-bar chart endpoint instead. An index that fails to load is left
// out rather than reported as zero, since a zero reads as a market-wide crash.
func (s *Server) GetMarketIndicesHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch the three concurrently - the endpoint takes one symbol per call
	quotes := make([]MarketIndex, len(marketIndices))
	failures := make([]error, len(marketIndices))

	var wg sync.WaitGroup
	for i, symbol := range marketIndices {
		wg.Go(func() {
			quotes[i], failures[i] = fetchIndexQuote(r.Context(), symbol)
		})
	}
	wg.Wait()

	// Keep whatever answered, and name what did not
	loaded := make([]MarketIndex, 0, len(quotes))
	var firstErr error
	for i, quote := range quotes {
		if failures[i] != nil {
			if firstErr == nil {
				firstErr = failures[i]
			}
			continue
		}
		loaded = append(loaded, quote)
	}

	if len(loaded) == 0 {
		slog.Error("[MarketIndex] no index could be loaded", "error", firstErr)
		common.WriteJSONError(w, http.StatusBadGateway, "Failed to fetch the market indices")
		return
	}
	if firstErr != nil {
		slog.Warn("[MarketIndex] some indices could not be loaded",
			"loaded", len(loaded), "of", len(marketIndices), "error", firstErr)
	}

	common.WriteJSON(w, http.StatusOK, loaded)
}

// fetchIndexQuote reads one index's last two daily bars and turns them into a
// quote: the latest close is where the index stands now, the one before it is
// the reference the session's change is measured from.
func fetchIndexQuote(ctx context.Context, symbol string) (MarketIndex, error) {
	body, err := json.Marshal(map[string]any{
		"timeFrame": "ONE_DAY",
		"symbols":   []string{symbol},
		"to":        time.Now().Unix(),
		"countBack": indexBars,
	})
	if err != nil {
		return MarketIndex{}, fmt.Errorf("marshal chart request for %s: %w", symbol, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chartURL, bytes.NewReader(body))
	if err != nil {
		return MarketIndex{}, fmt.Errorf("create chart request for %s: %w", symbol, err)
	}
	for k, v := range common.VCI_HEADERS {
		req.Header.Set(k, v)
	}

	resp, err := chartClient.Do(req)
	if err != nil {
		return MarketIndex{}, fmt.Errorf("fetch daily bars for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MarketIndex{}, fmt.Errorf("fetch daily bars for %s: unexpected status %d", symbol, resp.StatusCode)
	}

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return MarketIndex{}, fmt.Errorf("decompress daily bars for %s: %w", symbol, err)
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		return MarketIndex{}, fmt.Errorf("read daily bars for %s: %w", symbol, err)
	}

	var series []struct {
		Open   []float64 `json:"o"`
		High   []float64 `json:"h"`
		Low    []float64 `json:"l"`
		Close  []float64 `json:"c"`
		Volume []float64 `json:"v"`
	}
	if err := json.Unmarshal(respBytes, &series); err != nil {
		return MarketIndex{}, fmt.Errorf("decode daily bars for %s: %w", symbol, err)
	}
	if len(series) == 0 || len(series[0].Close) == 0 {
		return MarketIndex{}, fmt.Errorf("no daily bars for %s", symbol)
	}

	return indexQuoteFromBars(symbol, series[0].Open, series[0].High, series[0].Low, series[0].Close, series[0].Volume), nil
}

// indexQuoteFromBars builds the quote out of parallel bar series.
//
// The previous session's close is the reference; on the one day of history the
// endpoint sometimes returns, the session's own open stands in for it, which is
// what the exchange itself does for a newly listed index.
func indexQuoteFromBars(symbol string, opens, highs, lows, closes, volumes []float64) MarketIndex {
	latest := len(closes) - 1

	reference := closes[latest]
	if latest > 0 {
		reference = closes[latest-1]
	} else if len(opens) > 0 {
		reference = opens[latest]
	}

	quote := MarketIndex{Symbol: symbol, Value: closes[latest], Reference: reference}
	if latest < len(highs) {
		quote.High = highs[latest]
	}
	if latest < len(lows) {
		quote.Low = lows[latest]
	}
	if latest < len(volumes) {
		quote.Volume = volumes[latest]
	}

	// A zero reference would make the percentage meaningless, so leave it at zero
	if reference > 0 {
		quote.Change = quote.Value - reference
		quote.ChangePercent = quote.Change / reference * 100
	}

	return quote
}
