package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"stockmind/internal/common"
)

type GetStockPriceInput struct {
	Symbol    string `json:"symbol" jsonschema:"Stock symbol, e.g., HPG"`
	TimeFrame string `json:"time_frame,omitempty" jsonschema:"Time frame: ONE_DAY, ONE_MINUTE, or ONE_HOUR. Default is ONE_DAY"`
	CountBack int    `json:"count_back,omitempty" jsonschema:"Number of data points to look back. Default is 10"`
}

func HandleGetStockPrice(ctx context.Context, _ Deps, input GetStockPriceInput) (any, error) {
	if input.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	timeFrame := input.TimeFrame
	if timeFrame == "" {
		timeFrame = "ONE_DAY"
	}
	countBack := input.CountBack
	if countBack == 0 {
		countBack = 10
	}

	reqBody, _ := json.Marshal(map[string]any{
		"timeFrame": timeFrame,
		"symbols":   []string{input.Symbol},
		"to":        time.Now().Unix(),
		"countBack": countBack,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", common.TRADING_URL+common.CHART_URL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range common.VCI_HEADERS {
		httpReq.Header.Set(k, v)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fetch stock price: %w", err)
	}
	defer resp.Body.Close()

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, fmt.Errorf("decompress response: %w", err)
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var priceData []struct {
		Symbol string    `json:"symbol"`
		Open   []float64 `json:"o"`
		High   []float64 `json:"h"`
		Low    []float64 `json:"l"`
		Close  []float64 `json:"c"`
		Volume []int64   `json:"v"`
		Time   []string  `json:"t"`
	}
	if err := json.Unmarshal(respBytes, &priceData); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(priceData) == 0 {
		return nil, fmt.Errorf("no price data found for %s", input.Symbol)
	}

	data := priceData[0]
	type priceItem struct {
		Time   string  `json:"time"`
		Open   float64 `json:"open"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Close  float64 `json:"close"`
		Volume int64   `json:"volume"`
	}

	prices := make([]priceItem, 0, len(data.Time))
	for i := range data.Time {
		unixTime, err := strconv.ParseInt(data.Time[i], 10, 64)
		if err != nil {
			continue
		}
		if timeFrame != "ONE_DAY" {
			unixTime += 7 * 3600
		}
		prices = append(prices, priceItem{
			Time:   time.Unix(unixTime, 0).Format("2006-01-02 15:04:05"),
			Open:   data.Open[i],
			High:   data.High[i],
			Low:    data.Low[i],
			Close:  data.Close[i],
			Volume: data.Volume[i],
		})
	}

	return map[string]any{"symbol": data.Symbol, "prices": prices}, nil
}
