package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"stockmind/internal/agent/prompts"
	"stockmind/internal/common"
	"stockmind/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Configuration constants
// ---------------------------------------------------------------------------

const (
	tavilyPollInterval = 2 * time.Second
	tavilyPollTimeout  = 2 * time.Minute
	tavilyHTTPTimeout  = 30 * time.Second
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// ResearchRequest is the inbound payload for the research endpoint.
type ResearchRequest struct {
	Tickers       []string `json:"tickers"        validate:"required,min=1,max=5"`
	ResearchModel string   `json:"research_model" validate:"required,oneof=mini pro"`
}

// ---------------------------------------------------------------------------
// Tavily API types (typed structs instead of map[string]interface{})
// ---------------------------------------------------------------------------

// tavilyResearchRequest is the POST body sent to Tavily /research.
type tavilyResearchRequest struct {
	Input        string      `json:"input"`
	Model        string      `json:"model"`
	Stream       bool        `json:"stream"`
	OutputSchema interface{} `json:"output_schema,omitempty"`
}

// tavilyInitResponse is the initial async response from Tavily.
type tavilyInitResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
}

// tavilyPollResponse is the polling response. When status == "completed",
// Content will be populated.
type tavilyPollResponse struct {
	Status  string          `json:"status"`
	Content json.RawMessage `json:"content"`
	Sources []tavilySource  `json:"sources"`
}

// tavilySource is a source object in Tavily's top-level response.
type tavilySource struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ---------------------------------------------------------------------------
// Tavily client (encapsulates HTTP + auth)
// ---------------------------------------------------------------------------

type tavilyClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func newTavilyClient() *tavilyClient {
	return &tavilyClient{
		httpClient: &http.Client{Timeout: tavilyHTTPTimeout},
		baseURL:    common.TAVILY_URL,
		apiKey:     os.Getenv("TAVILY_API_KEY"),
	}
}

// submitResearch kicks off an async research job and returns the request ID.
func (tc *tavilyClient) submitResearch(ctx context.Context, payload tavilyResearchRequest) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal tavily request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tc.baseURL+"/research", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create tavily request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tc.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := tc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("tavily POST /research: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("tavily POST /research returned %d: %s", res.StatusCode, string(respBody))
	}

	var initResp tavilyInitResponse
	if err := json.NewDecoder(res.Body).Decode(&initResp); err != nil {
		return "", fmt.Errorf("decode tavily init response: %w", err)
	}

	if initResp.RequestID == "" {
		return "", fmt.Errorf("tavily returned empty request_id (status=%s)", initResp.Status)
	}

	return initResp.RequestID, nil
}

// pollResult polls Tavily until the research is "completed", "failed", or the context is cancelled.
func (tc *tavilyClient) pollResult(ctx context.Context, requestID string) (*tavilyPollResponse, error) {
	ticker := time.NewTicker(tavilyPollInterval)
	defer ticker.Stop()

	deadline := time.After(tavilyPollTimeout)

	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("tavily poll timed out after %v", tavilyPollTimeout)
		case <-ticker.C:
			// continue polling below
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, tc.baseURL+"/research/"+requestID, nil)
		if err != nil {
			return nil, fmt.Errorf("create poll request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tc.apiKey)

		res, err := tc.httpClient.Do(req)
		if err != nil {
			slog.Warn("tavily poll error, retrying", "attempt", attempt, "error", err)
			continue
		}

		var pollResp tavilyPollResponse
		if decErr := json.NewDecoder(res.Body).Decode(&pollResp); decErr != nil {
			res.Body.Close()
			slog.Warn("tavily poll decode error, retrying", "attempt", attempt, "error", decErr)
			continue
		}
		res.Body.Close()

		slog.Info("tavily poll", "attempt", attempt, "status", pollResp.Status, "request_id", requestID)

		switch pollResp.Status {
		case "completed":
			return &pollResp, nil
		case "failed", "error":
			return nil, fmt.Errorf("tavily research failed (request_id=%s)", requestID)
		}
		// status is "pending" / "in_progress" → loop
	}
}

// ---------------------------------------------------------------------------
// Request validation (shared by both handlers)
// ---------------------------------------------------------------------------

func decodeAndValidateRequest(r *http.Request) (ResearchRequest, error) {
	var req ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("Invalid JSON")
	}
	if len(req.Tickers) == 0 {
		return req, fmt.Errorf("tickers must be a non-empty list")
	}
	if req.ResearchModel != "mini" && req.ResearchModel != "pro" {
		return req, fmt.Errorf("research_model must be 'mini' or 'pro'")
	}
	return req, nil
}

// ---------------------------------------------------------------------------
// SSE Progress types
// ---------------------------------------------------------------------------

type progressEvent struct {
	Ticker   string `json:"ticker"`
	Step     string `json:"step"`
	Message  string `json:"message"`
	Progress int    `json:"progress"` // 0-100 per ticker
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// MarketResearchHandler is the synchronous (non-streaming) research endpoint.
func (s *Server) MarketResearchHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateRequest(r)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := stockDigestAgent(r.Context(), req, nil)
	common.WriteJSON(w, http.StatusOK, response)
}

// MarketResearchStreamHandler is the SSE streaming research endpoint.
func (s *Server) MarketResearchStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	req, err := decodeAndValidateRequest(r)
	if err != nil {
		writeSSE(w, map[string]any{"type": "error", "data": map[string]any{"message": err.Error()}})
		return
	}

	progressCh := make(chan progressEvent, 50)
	doneCh := make(chan database.ResearchReport, 1)

	go func() {
		doneCh <- stockDigestAgent(r.Context(), req, progressCh)
	}()

	for evt := range progressCh {
		writeSSE(w, map[string]any{"type": "progress", "data": evt})

		// Check if client disconnected between events
		if r.Context().Err() != nil {
			return
		}
	}

	// All progress drained, send the final result
	writeSSE(w, map[string]any{"type": "result", "data": <-doneCh})
}

// ---------------------------------------------------------------------------
// Orchestrator — researches tickers concurrently.
// ---------------------------------------------------------------------------

func stockDigestAgent(ctx context.Context, request ResearchRequest, progressCh chan<- progressEvent) database.ResearchReport {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		reports = make(map[string]database.StockReport, len(request.Tickers))
	)

	tc := newTavilyClient()

	for _, ticker := range request.Tickers {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()

			report, err := researchTicker(ctx, tc, t, request.ResearchModel, progressCh)
			if err != nil {
				slog.Error("research failed", "ticker", t, "error", err)
				return
			}

			mu.Lock()
			reports[t] = report
			mu.Unlock()
		}(ticker)
	}

	wg.Wait()
	if progressCh != nil {
		close(progressCh)
	}

	return database.ResearchReport{
		Reports:     reports,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// Single-ticker research with optional progress reporting
// ---------------------------------------------------------------------------

func researchTicker(ctx context.Context, tc *tavilyClient, ticker, model string, progressCh chan<- progressEvent) (database.StockReport, error) {
	emit := func(step, message string, progress int) {
		if progressCh == nil {
			return
		}
		progressCh <- progressEvent{
			Ticker:   ticker,
			Step:     step,
			Message:  message,
			Progress: progress,
		}
	}

	// 1. Build the prompt
	emit("building_prompt", "Building research prompt...", 10)
	loader := prompts.NewPromptLoader()
	prompt, err := loader.GetResearchPrompt(prompts.ResearchParams{
		Ticker: ticker,
		Date:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		emit("failed", fmt.Sprintf("Failed to build prompt: %v", err), 0)
		return database.StockReport{}, fmt.Errorf("build prompt for %s: %w", ticker, err)
	}

	// 2. Submit async research job
	emit("submitting", "Submitting research request...", 25)
	requestID, err := tc.submitResearch(ctx, tavilyResearchRequest{
		Input:        prompt,
		Model:        model,
		Stream:       false,
		OutputSchema: outputSchema(),
	})
	if err != nil {
		emit("failed", fmt.Sprintf("Failed to submit: %v", err), 0)
		return database.StockReport{}, fmt.Errorf("submit research for %s: %w", ticker, err)
	}
	slog.Info("tavily research submitted", "ticker", ticker, "request_id", requestID)

	// 3. Poll until completed
	emit("polling", "Gathering and analyzing data...", 40)
	pollResp, err := tc.pollResult(ctx, requestID)
	if err != nil {
		emit("failed", fmt.Sprintf("Research timed out: %v", err), 0)
		return database.StockReport{}, fmt.Errorf("poll research for %s: %w", ticker, err)
	}

	// 4. Parse structured content
	emit("parsing", "Parsing research results...", 85)
	report, err := parseResearchResult(ticker, pollResp)
	if err != nil {
		emit("failed", fmt.Sprintf("Failed to parse: %v", err), 0)
		return database.StockReport{}, err
	}

	emit("completed", "Research complete", 100)
	return report, nil
}

// ---------------------------------------------------------------------------
// Output schema (defined once, reused for every request)
// ---------------------------------------------------------------------------

func outputSchema() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"ticker": map[string]interface{}{
				"type":        "string",
				"description": "The official stock ticker symbol used on exchanges (e.g., AAPL, GOOGL, MSFT)",
			},
			"company_name": map[string]interface{}{
				"type":        "string",
				"description": "The full legal or commonly used name of the company",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "A comprehensive overview of the stock analysis including recent developments, market position, and overall assessment",
			},
			"current_performance": map[string]interface{}{
				"type":        "string",
				"description": "Detailed analysis of recent stock performance including price movements, trading patterns, and comparison to market benchmarks",
			},
			"key_insights": map[string]interface{}{
				"type":        "array",
				"description": "Critical takeaways and notable observations from trusted financial analysts and market experts",
				"items":       map[string]interface{}{"type": "string"},
			},
			"recommendation": map[string]interface{}{
				"type":        "string",
				"description": "Investment recommendation such as buy, hold, or sell, along with supporting rationale and target audience considerations",
			},
			"risk_assessment": map[string]interface{}{
				"type":        "string",
				"description": "Evaluation of potential risks including market volatility, company-specific challenges, regulatory concerns, and macroeconomic factors",
			},
			"price_outlook": map[string]interface{}{
				"type":        "string",
				"description": "Forward-looking analysis of expected price movements including short-term and long-term projections with supporting factors",
			},
			"market_cap": map[string]interface{}{
				"type":        "string",
				"description": "Total market capitalization in US dollars, representing the company's total market value of outstanding shares",
			},
			"pe_ratio": map[string]interface{}{
				"type":        "string",
				"description": "Price-to-earnings ratio indicating how much investors are willing to pay per dollar of earnings",
			},
			"sources": map[string]interface{}{
				"type":        "array",
				"description": "List of referenced sources including news articles, analyst reports, and financial publications used in the analysis",
				"items":       map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"ticker", "company_name", "summary", "current_performance", "key_insights", "recommendation", "risk_assessment", "price_outlook", "sources"},
	}
}

// ---------------------------------------------------------------------------
// Response parsing
// ---------------------------------------------------------------------------

type tavilyStructuredContent struct {
	Ticker             string   `json:"ticker"`
	CompanyName        string   `json:"company_name"`
	Summary            string   `json:"summary"`
	CurrentPerformance string   `json:"current_performance"`
	KeyInsights        []string `json:"key_insights"`
	Recommendation     string   `json:"recommendation"`
	RiskAssessment     string   `json:"risk_assessment"`
	PriceOutlook       string   `json:"price_outlook"`
	MarketCap          string   `json:"market_cap"`
	PERatio            string   `json:"pe_ratio"`
	Sources            []string `json:"sources"`
}

func parseResearchResult(ticker string, resp *tavilyPollResponse) (database.StockReport, error) {
	report := database.StockReport{Ticker: ticker}

	// Try to unmarshal structured content
	if len(resp.Content) > 0 {
		var structured tavilyStructuredContent
		if err := json.Unmarshal(resp.Content, &structured); err == nil {
			report.CompanyName = structured.CompanyName
			report.Summary = structured.Summary
			report.CurrentPerformance = structured.CurrentPerformance
			report.KeyInsights = structured.KeyInsights
			report.Recommendation = structured.Recommendation
			report.RiskAssessment = structured.RiskAssessment
			report.PriceOutlook = structured.PriceOutlook
			report.MarketCap = structured.MarketCap
			report.PERatio = structured.PERatio

			// Convert string sources into Source objects
			for _, s := range structured.Sources {
				report.Sources = append(report.Sources, database.Source{URL: s})
			}
		} else {
			// Fall back: content is a plain string summary.
			var plainSummary string
			if json.Unmarshal(resp.Content, &plainSummary) == nil {
				report.Summary = plainSummary
			}
		}
	}

	// Merge top-level Tavily sources (these carry titles).
	if len(resp.Sources) > 0 && len(report.Sources) == 0 {
		for _, s := range resp.Sources {
			report.Sources = append(report.Sources, database.Source{
				URL:   s.URL,
				Title: s.Title,
			})
		}
	}

	return report, nil
}

func (s *Server) GetResearchReportsHandler(w http.ResponseWriter, r *http.Request) {
	reports, err := s.db.GetResearchReports(r.Context())
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, reports)
}

func (s *Server) GetResearchReportByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	report, err := s.db.GetResearchReportById(r.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, report)
}
