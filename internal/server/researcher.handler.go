package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/prompts"
	"stockmind/internal/service/tavily"
	"stockmind/internal/service/worker"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Domain types (inbound request, shared across handlers)
// ---------------------------------------------------------------------------

// ResearchRequestPayload is the inbound payload for the research endpoint.
type ResearchRequestPayload struct {
	Tickers       []string `json:"tickers"        validate:"required,min=1,max=5"`
	ResearchModel string   `json:"research_model" validate:"required,oneof=mini pro"`
}

// ---------------------------------------------------------------------------
// Request validation (shared by both handlers)
// ---------------------------------------------------------------------------

func decodeAndValidateRequest(r *http.Request) (ResearchRequestPayload, error) {
	var req ResearchRequestPayload
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
// Helpers
// ---------------------------------------------------------------------------

// firstWord returns the first whitespace-delimited word from s.
// If s is empty or blank, it returns "Unknown".
func firstWord(s string) string {
	if words := strings.Fields(s); len(words) > 0 {
		return words[0]
	}
	return "Unknown"
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

	response := s.stockDigestAgent(r.Context(), req, nil)
	go s.persistReports(response)
	common.WriteJSON(w, http.StatusOK, response)
}

// MarketResearchStreamHandler is the SSE streaming research endpoint.
func (s *Server) MarketResearchStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	req, err := decodeAndValidateRequest(r)
	if err != nil {
		common.WriteSSE(w, map[string]any{"type": "error", "data": map[string]any{"message": err.Error()}})
		return
	}

	progressCh := make(chan progressEvent, 50)
	doneCh := make(chan database.ResearchReport, 1)

	go func() {
		doneCh <- s.stockDigestAgent(r.Context(), req, progressCh)
	}()

	for evt := range progressCh {
		common.WriteSSE(w, map[string]any{"type": "progress", "data": evt})

		// Check if client disconnected between events
		if r.Context().Err() != nil {
			return
		}
	}

	// All progress drained, send the final result
	response := <-doneCh
	common.WriteSSE(w, map[string]any{"type": "result", "data": response})

	// Persist research reports in the background
	go s.persistReports(response)
}

// ---------------------------------------------------------------------------
// persistReports saves each report to the database in the background.
// ---------------------------------------------------------------------------

func (s *Server) persistReports(report database.ResearchReport) {
	for _, ticker := range report.Reports {
		price, err := GetLatestMatchPrice(context.Background(), ticker.Ticker)
		if err != nil {
			slog.Error("Failed to fetch stock price", "ticker", ticker.Ticker, "error", err)
		}
		_, err = s.queries.CreateResearchReport(context.Background(), database.CreateResearchReportParams{
			Ticker:         ticker.Ticker,
			Recommendation: firstWord(ticker.Recommendation),
			ReferencePrice: strconv.FormatFloat(price, 'f', -1, 64),
			Report:         ticker,
		})
		if err != nil {
			slog.Error("Failed to create research report", "ticker", ticker.Ticker, "error", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Orchestrator — researches tickers via the worker pool.
// ---------------------------------------------------------------------------

func (s *Server) stockDigestAgent(ctx context.Context, request ResearchRequestPayload, progressCh chan<- progressEvent) database.ResearchReport {
	resultCh := make(chan worker.ResearchResult, len(request.Tickers))

	var progressAnyCh chan<- any
	if progressCh != nil {
		proxy := make(chan any, 50)
		progressAnyCh = proxy
		go func() {
			for v := range proxy {
				if evt, ok := v.(progressEvent); ok {
					progressCh <- evt
				}
			}
		}()
	}

	for _, ticker := range request.Tickers {
		job := &worker.ResearchJob{
			Ticker:        ticker,
			ResearchModel: request.ResearchModel,
			ProgressCh:    progressAnyCh,
			ResultCh:      resultCh,
		}
		if err := s.services.ResearchWorker.Enqueue(job); err != nil {
			slog.Error("research enqueue failed", "ticker", ticker, "error", err)
			resultCh <- worker.ResearchResult{Ticker: ticker, Err: err}
		}
	}

	reports := make(map[string]database.StockReport, len(request.Tickers))
	for range request.Tickers {
		r := <-resultCh
		if r.Err != nil {
			slog.Error("research failed", "ticker", r.Ticker, "error", r.Err)
			continue
		}
		reports[r.Ticker] = r.Report
	}

	if progressAnyCh != nil {
		close(progressAnyCh)
	}
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

// ProcessResearchJob is the worker process function for research jobs.
func (s *Server) ProcessResearchJob(job *worker.ResearchJob) {
	// Build a typed progress channel from the any channel
	var progressCh chan<- progressEvent
	if job.ProgressCh != nil {
		progressCh = newProgressAdapter(job.ProgressCh)
	}

	report, err := s.researchTicker(context.Background(), job.Ticker, job.ResearchModel, progressCh)
	job.ResultCh <- worker.ResearchResult{Ticker: job.Ticker, Report: report, Err: err}
}

// newProgressAdapter converts chan<- any to chan<- progressEvent via a goroutine.
func newProgressAdapter(ch chan<- any) chan<- progressEvent {
	typed := make(chan progressEvent, 50)
	go func() {
		for evt := range typed {
			ch <- evt
		}
	}()
	return typed
}

func (s *Server) researchTicker(ctx context.Context, ticker, model string, progressCh chan<- progressEvent) (database.StockReport, error) {
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
	requestID, err := s.services.Tavily.SubmitResearch(ctx, tavily.ResearchRequest{
		Input:        prompt,
		Model:        model,
		Stream:       false,
		OutputSchema: tavily.ResearchOutputSchema(),
	})
	if err != nil {
		emit("failed", fmt.Sprintf("Failed to submit: %v", err), 0)
		return database.StockReport{}, fmt.Errorf("submit research for %s: %w", ticker, err)
	}
	slog.Info("tavily research submitted", "ticker", ticker, "request_id", requestID)

	// 3. Poll until completed
	emit("polling", "Gathering and analyzing data...", 40)
	pollResp, err := s.services.Tavily.PollResearch(ctx, requestID)
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
// Response parsing — bridges tavily response to domain types
// ---------------------------------------------------------------------------

func parseResearchResult(ticker string, resp *tavily.ResearchPollResponse) (database.StockReport, error) {
	report := database.StockReport{Ticker: ticker}

	if len(resp.Content) == 0 {
		return report, nil
	}

	// Try structured JSON first — unmarshals directly into StockReport.
	// Source's custom UnmarshalJSON handles both plain strings and objects.
	if err := json.Unmarshal(resp.Content, &report); err != nil {
		// Fallback: content may be a plain string summary.
		var plainSummary string
		if err := json.Unmarshal(resp.Content, &plainSummary); err == nil {
			report.Summary = plainSummary
			return report, nil
		}
		// Neither structured nor string — return what we have.
		return report, nil
	}

	// Ensure ticker is set (the AI response may omit it or use a different casing).
	report.Ticker = ticker

	// Merge top-level Tavily sources (these carry titles) if the AI didn't provide any.
	if len(resp.Sources) > 0 && len(report.Sources) == 0 {
		for _, src := range resp.Sources {
			report.Sources = append(report.Sources, database.Source{
				URL:   src.URL,
				Title: src.Title,
			})
		}
	}

	return report, nil
}

// ---------------------------------------------------------------------------
// Report retrieval handlers
// ---------------------------------------------------------------------------

type ResearchReport struct {
	database.GetResearchReportsRow
	Price float64 `json:"price"`
}

func (s *Server) GetResearchReportsHandler(w http.ResponseWriter, r *http.Request) {
	var reports []ResearchReport
	reportFromDB, err := s.queries.GetResearchReports(r.Context())
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, report := range reportFromDB {
		price, err := GetLatestMatchPrice(context.Background(), report.Ticker)
		if err != nil {
			common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		reports = append(reports, ResearchReport{
			GetResearchReportsRow: report,
			Price:                 price,
		})
	}
	common.WriteJSON(w, http.StatusOK, reports)
}

func (s *Server) GetResearchReportByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	report, err := s.queries.GetResearchReportById(r.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, report)
}
