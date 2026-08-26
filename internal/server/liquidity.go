package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"stockmind/internal/common"
)

const (
	// minAverageVolume is the average daily volume a ticker has to trade to earn a
	// place on a sector board. Below it the quote is one nobody can act on: the
	// spread is wide, the last match may be days old, and an order of any size
	// moves the price.
	minAverageVolume = 100_000

	// averageVolumeWindow is the span of calendar time the average covers. It has
	// to be calendar time rather than a count of bars: the chart endpoint returns
	// the last N bars that exist, so a ticker the exchange suspended in March
	// still answers with March's bars and averages like a liquid stock.
	averageVolumeWindow = 30 * 24 * time.Hour

	// averageVolumeBars is how many bars to ask for - comfortably more than the
	// ~22 sessions a 30-day window holds, so the window is never bar-limited.
	averageVolumeBars = 40

	// averageVolumeTTL is how long a computed average is reused. A twenty-session
	// average barely moves in a day, and it costs one request per ticker to build.
	averageVolumeTTL = 12 * time.Hour

	// averageVolumeConcurrency bounds the fan-out. VietCap's chart endpoint takes
	// exactly one symbol per call - a batch of symbols answers with an empty array
	// rather than an error - so the largest industry is over 300 requests.
	averageVolumeConcurrency = 16
)

// chartURL is the daily-bar endpoint, indirected through a var so tests can
// point it at a stub server. Nothing outside this package writes it.
var chartURL = common.TRADING_URL + common.CHART_URL

// chartClient reuses connections across that fan-out. Against the default
// client a fresh TLS handshake per ticker dominates the whole operation.
var chartClient = &http.Client{
	Timeout: vciHTTPTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        averageVolumeConcurrency,
		MaxIdleConnsPerHost: averageVolumeConcurrency,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}

// tickerVolume is one ticker's average and the moment it was computed.
type tickerVolume struct {
	average   float64
	fetchedAt time.Time
}

// averageVolumeCache holds average daily volumes behind a TTL. Entries are keyed
// by ticker rather than by industry, so a ticker priced for one sector is already
// warm for every other sector and for the next request. The zero value is ready
// to use.
type averageVolumeCache struct {
	mu       sync.Mutex
	byTicker map[string]tickerVolume
}

// averages returns the average daily volume of every symbol it could price.
//
// A symbol VietCap has no history for is left out of the result rather than
// recorded as a zero, so a failed lookup can never read as an illiquid stock.
func (c *averageVolumeCache) averages(ctx context.Context, symbols []string) map[string]float64 {
	// Serve what the cache already holds and collect the rest
	known := make(map[string]float64, len(symbols))
	missing := make([]string, 0, len(symbols))

	c.mu.Lock()
	for _, symbol := range symbols {
		if entry, ok := c.byTicker[symbol]; ok && time.Since(entry.fetchedAt) < averageVolumeTTL {
			known[symbol] = entry.average
			continue
		}
		missing = append(missing, symbol)
	}
	c.mu.Unlock()

	if len(missing) == 0 {
		return known
	}

	// Price the rest concurrently, bounded so a large industry does not flood VietCap
	fetched := make([]struct {
		average float64
		err     error
	}, len(missing))

	slots := make(chan struct{}, averageVolumeConcurrency)
	var wg sync.WaitGroup
	for i, symbol := range missing {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()

			fetched[i].average, fetched[i].err = fetchAverageVolume(ctx, symbol)
		})
	}
	wg.Wait()

	// Cache and merge whatever came back
	now := time.Now()
	var firstErr error
	unpriced := 0

	c.mu.Lock()
	if c.byTicker == nil {
		c.byTicker = make(map[string]tickerVolume, len(symbols))
	}
	for i, symbol := range missing {
		if fetched[i].err != nil {
			unpriced++
			if firstErr == nil {
				firstErr = fetched[i].err
			}
			continue
		}
		c.byTicker[symbol] = tickerVolume{average: fetched[i].average, fetchedAt: now}
		known[symbol] = fetched[i].average
	}
	c.mu.Unlock()

	if unpriced > 0 {
		slog.Warn("[Liquidity] could not price some tickers; they stay on the board",
			"unpriced", unpriced, "priced", len(missing)-unpriced, "error", firstErr)
	}

	return known
}

// dropIlliquidRows removes rows whose average daily volume is under the board's
// minimum.
//
// A row with no average is kept. A history lookup that failed is not evidence
// that a stock is illiquid, and silently hiding a ticker because one request
// timed out is worse than showing a thin one.
func dropIlliquidRows(rows []SectorBoardRow, averages map[string]float64) []SectorBoardRow {
	liquid := make([]SectorBoardRow, 0, len(rows))
	for _, row := range rows {
		if average, ok := averages[row.ListingInfo.Symbol]; ok && average < minAverageVolume {
			continue
		}
		liquid = append(liquid, row)
	}
	return liquid
}

// fetchAverageVolume averages one symbol's traded volume across the last
// averageVolumeWindow of calendar time.
//
// Sessions the symbol did not trade count as zero rather than being left out of
// the divisor. A ticker under suspension, or one restricted to a single session
// a week, is illiquid now whatever it traded before, and averaging only over the
// sessions it did trade would report it as liquid - which is the whole thing
// this filter exists to catch.
func fetchAverageVolume(ctx context.Context, symbol string) (float64, error) {
	body, err := json.Marshal(map[string]any{
		"timeFrame": "ONE_DAY",
		"symbols":   []string{symbol},
		"to":        time.Now().Unix(),
		"countBack": averageVolumeBars,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal chart request for %s: %w", symbol, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chartURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create chart request for %s: %w", symbol, err)
	}
	for k, v := range common.VCI_HEADERS {
		req.Header.Set(k, v)
	}

	resp, err := chartClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch daily bars for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetch daily bars for %s: unexpected status %d", symbol, resp.StatusCode)
	}

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return 0, fmt.Errorf("decompress daily bars for %s: %w", symbol, err)
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		return 0, fmt.Errorf("read daily bars for %s: %w", symbol, err)
	}

	// The chart endpoint returns one entry per symbol, carrying parallel series
	// of bar timestamps (unix seconds, as strings) and volumes
	var series []struct {
		Time   []string  `json:"t"`
		Volume []float64 `json:"v"`
	}
	if err := json.Unmarshal(respBytes, &series); err != nil {
		return 0, fmt.Errorf("decode daily bars for %s: %w", symbol, err)
	}
	if len(series) == 0 || len(series[0].Volume) == 0 {
		return 0, fmt.Errorf("no daily bars for %s", symbol)
	}
	if len(series[0].Time) != len(series[0].Volume) {
		return 0, fmt.Errorf("daily bars for %s carry %d timestamps for %d volumes",
			symbol, len(series[0].Time), len(series[0].Volume))
	}

	// Total only what traded inside the window
	windowStart := time.Now().Add(-averageVolumeWindow)
	total := 0.0
	for i, volume := range series[0].Volume {
		at, err := strconv.ParseInt(series[0].Time[i], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("decode bar timestamp %q for %s: %w", series[0].Time[i], symbol, err)
		}
		if time.Unix(at, 0).After(windowStart) {
			total += volume
		}
	}

	sessions := tradingSessionsSince(windowStart)
	if sessions == 0 {
		return 0, fmt.Errorf("no trading sessions in the last %v", averageVolumeWindow)
	}

	return total / float64(sessions), nil
}

// tradingSessionsSince counts the weekdays between the given time and now. It
// stands in for the exchange calendar, which VietCap does not publish; counting
// a public holiday as a session understates every ticker equally, so it does not
// change which side of the threshold one lands on.
func tradingSessionsSince(start time.Time) int {
	sessions := 0
	for day := start; day.Before(time.Now()); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			sessions++
		}
	}
	return sessions
}
