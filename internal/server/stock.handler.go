package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stockmind/internal/common"
	"time"
)

type StockPrice struct {
	Symbol string `json:"symbol"`
	Prices string `json:"prices"`
}

func (s *Server) GetPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbols []string `json:"symbols"`
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	// Parse JSON request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	body, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Create new request
	// FIX: Use PRICE_BOARD_URL instead of CHART_URL
	http_req, err := http.NewRequestWithContext(r.Context(), "POST", fmt.Sprintf("%s%s", common.TRADING_URL, common.PRICE_BOARD_URL), bytes.NewBuffer(body))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Write headers from VCI_HEADERS
	for k, v := range common.VCI_HEADERS {
		http_req.Header.Set(k, v)
	}

	// Fetch stock price from VCI
	resp, err := client.Do(http_req)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch stock prices")
		return
	}
	defer resp.Body.Close()

	// Decompress if needed
	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Decompression failed")
		return
	}
	defer reader.Close()

	respBytes, err := io.ReadAll(reader)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// Decode into generic interface to see original response structure
	var priceBoard interface{}
	if err := json.Unmarshal(respBytes, &priceBoard); err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Invalid JSON response from upstream provider")
		return
	}

	// Write response
	common.WriteJSON(w, http.StatusOK, priceBoard)
}
