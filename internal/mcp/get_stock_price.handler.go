package mcp

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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VCITimeFrame string

const (
	ONE_DAY    VCITimeFrame = "ONE_DAY"
	ONE_MINUTE VCITimeFrame = "ONE_MINUTE"
	ONE_HOUR   VCITimeFrame = "ONE_HOUR"
)

type VCIStockRequest struct {
	TimeFrame VCITimeFrame `json:"timeFrame"`
	Symbols   []string     `json:"symbols"`
	To        int64        `json:"to"`
	CountBack int32        `json:"countBack"`
}

type vciPriceDataResponse struct {
	Symbol            string    `json:"symbol"`
	Open              []float64 `json:"o"`
	High              []float64 `json:"h"`
	Low               []float64 `json:"l"`
	Close             []float64 `json:"c"`
	Volume            []int64   `json:"v"`
	Time              []string  `json:"t"`
	AccumulatedVolume []int64   `json:"accumulatedVolume,omitempty"`
	AccumulatedValue  []float64 `json:"accumulatedValue,omitempty"`
	MinBatchTruncTime string    `json:"minBatchTruncTime"`
}

type StockPriceItem struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

type StockPrice struct {
	Symbol string           `json:"symbol"`
	Prices []StockPriceItem `json:"prices"`
}

type GetStockPriceInput struct {
	Symbol    string `json:"symbol" jsonschema:"Stock symbol, e.g., HPG"`
	TimeFrame string `json:"time_frame" jsonschema:"Time frame, e.g., ONE_DAY, ONE_MINUTE, ONE_HOUR. Default is ONE_DAY"`
	CountBack int    `json:"count_back" jsonschema:"Number of data points to look back. Default is 10"`
}

func GetStockPrice(ctx context.Context, req *mcp.CallToolRequest, input GetStockPriceInput) (*mcp.CallToolResult, StockPrice, error) {
	symbol := input.Symbol
	if symbol == "" {
		return nil, StockPrice{}, fmt.Errorf("symbol is required")
	}
	timeFrame := VCITimeFrame(input.TimeFrame)
	if timeFrame == "" {
		timeFrame = ONE_DAY
	}
	countBack := input.CountBack
	if countBack == 0 {
		countBack = 10
	}

	stockRequest := VCIStockRequest{
		TimeFrame: timeFrame,
		Symbols:   []string{symbol},
		To:        time.Now().Unix(),
		CountBack: int32(countBack),
	}
	client := &http.Client{Timeout: 10 * time.Second}
	body, err := json.Marshal(stockRequest)
	if err != nil {
		return nil, StockPrice{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s%s", common.TRADING_URL, common.CHART_URL), bytes.NewBuffer(body))
	if err != nil {
		return nil, StockPrice{}, err
	}
	for k, v := range common.VCI_HEADERS {
		httpReq.Header.Set(k, v)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, StockPrice{}, err
	}
	defer resp.Body.Close()

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, StockPrice{}, fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, StockPrice{}, fmt.Errorf("failed to read response body: %w", err)
	}
	var priceData []vciPriceDataResponse
	if err := json.Unmarshal(respBytes, &priceData); err != nil {
		return nil, StockPrice{}, fmt.Errorf("failed to unmarshal response: %w, rawresponse: %s", err, string(respBytes))
	}
	if len(priceData) == 0 {
		return nil, StockPrice{}, fmt.Errorf("no price data found")
	}
	data := priceData[0]
	prices := make([]StockPriceItem, 0, len(data.Time))
	for i := range data.Time {
		unixTime, err := strconv.ParseInt(data.Time[i], 10, 64)
		if err != nil {
			return nil, StockPrice{}, fmt.Errorf("invalid time format: %w", err)
		}
		if timeFrame != ONE_DAY {
			unixTime += 7 * 3600
		}
		prices = append(prices, StockPriceItem{
			Time:   time.Unix(unixTime, 0),
			Open:   data.Open[i],
			High:   data.High[i],
			Low:    data.Low[i],
			Close:  data.Close[i],
			Volume: data.Volume[i],
		})
	}
	result := StockPrice{Symbol: data.Symbol, Prices: prices}
	return nil, result, nil
}
