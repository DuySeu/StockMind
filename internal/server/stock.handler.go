package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"stockmind/internal/common"
	"time"
)

type AddSymbolInPriceBoardRequest struct {
	Symbol string `json:"symbol"`
}

func (s *Server) GetPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	// Get symbols from watchlist in database
	watchlist, err := s.db.GetWatchlist(r.Context())
	if err != nil {
		log.Printf("[PriceBoard] Failed to get watchlist: %v", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get watchlist")
		return
	}

	symbols := make([]string, 0, len(watchlist))
	for _, item := range watchlist {
		symbols = append(symbols, item.Ticker)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	body, err := json.Marshal(struct {
		Symbols []string `json:"symbols"`
	}{Symbols: symbols})
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to marshal symbols")
		return
	}
	// Create new request
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

func (s *Server) AddSymbolInPriceBoardHandler(w http.ResponseWriter, r *http.Request) {
	var req AddSymbolInPriceBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add symbol to watchlist
	_, err := s.db.CreateWatchlistData(r.Context(), req.Symbol)
	if err != nil {
		log.Printf("[PriceBoard] Failed to add symbol to watchlist: %v", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to add symbol to watchlist")
		return
	}

	// Write response
	common.WriteJSON(w, http.StatusCreated, "Symbol added to watchlist successfully")
}

func (s *Server) GetWatchlistHandler(w http.ResponseWriter, r *http.Request) {
	watchlist, err := s.db.GetWatchlist(r.Context())
	if err != nil {
		log.Printf("[PriceBoard] Failed to get watchlist: %v", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get watchlist")
		return
	}

	common.WriteJSON(w, http.StatusOK, watchlist)
}
