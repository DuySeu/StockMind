package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/prompts"
	"stockmind/internal/service/tavily"
	"stockmind/internal/service/worker"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ResearchRequestPayload struct {
	Tickers       []string `json:"tickers"        validate:"required,min=1,max=5"`
	ResearchModel string   `json:"research_model" validate:"required,oneof=mini pro"`
}

type FundamentalAnalysisRequestPayload struct {
	Symbol string `json:"symbol" validate:"required"`
}

type ResearchReport struct {
	database.GetResearchReportsRow
	Price float64 `json:"price"`
}

type FundamentalAnalysisOutput struct {
	Overview         Overview         `json:"overview"`
	BusinessActivity BusinessActivity `json:"business_activity"`
	EconomicMoat     string           `json:"economic_moat"`
	Outlook          Outlook          `json:"outlook"`
	Risks            Risks            `json:"risks"`
	Macro            string           `json:"macro"`
}

type Overview struct {
	CompanyName          string         `json:"company_name"`
	Industry             string         `json:"industry"`
	Brief                string         `json:"brief"`
	ShareholderStructure map[string]any `json:"shareholder_structure"`
	Ecosystem            map[string]any `json:"ecosystem,omitempty"`
}

type BusinessActivity struct {
	RevenueAndProfitStructure map[string]any `json:"revenue_and_profit_structure"`
	OperationalProcesses      string         `json:"operational_processes"`
	IndustrySpecific          string         `json:"industry_specific"`
}

type Risks struct {
	EconomicRisks    string   `json:"economic_risks"`
	LegalRisks       string   `json:"legal_risks"`
	DependentFactors []string `json:"dependent_factors"`
}

type Outlook struct {
	Potential      []string `json:"potential_factors"`
	ExpansionPlans string   `json:"expansion_plans"`
}

type progressEvent struct {
	Ticker   string `json:"ticker"`
	Step     string `json:"step"`
	Message  string `json:"message"`
	Progress int    `json:"progress"` // 0-100 per ticker
}

// POST /v1/stock/fundamental-analysis - Analyse one company's fundamentals
func (s *Server) FundamentalAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	var input FundamentalAnalysisRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	if symbol == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	ctx := r.Context()

	// Company details ground everything below, so this one is fatal.
	details, err := faFetchIQ[faCompanyDetails](ctx, fmt.Sprintf("%s/details?ticker=%s", common.COMPANY_URL, symbol))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("fetch company details for %s: %v", symbol, err))
		return
	}
	if details.ViOrganName == "" && details.EnOrganName == "" {
		common.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("no company data found for %s", symbol))
		return
	}

	// Shareholders and relationships are best-effort: a failure here leaves the
	// zero value in place rather than failing the request.
	var (
		wg      sync.WaitGroup
		holders faShareholderStructure
		rel     faRelationship
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		h, err := faFetchIQ[faShareholderStructure](ctx, fmt.Sprintf("%s/%s/shareholder-structure", common.COMPANY_URL, symbol))
		if err != nil {
			slog.Warn("fundamental_analysis: fetch shareholder structure", "symbol", symbol, "error", err)
			return
		}
		holders = h
	}()
	go func() {
		defer wg.Done()
		rl, err := faFetchIQ[faRelationship](ctx, fmt.Sprintf("%s/%s/relationship", common.COMPANY_URL, symbol))
		if err != nil {
			slog.Warn("fundamental_analysis: fetch relationship", "symbol", symbol, "error", err)
			return
		}
		rel = rl
	}()
	wg.Wait()

	result := FundamentalAnalysisOutput{
		Overview: Overview{
			CompanyName: faFirstNonEmpty(details.ViOrganName, details.EnOrganName),
			Industry:    faFirstNonEmpty(details.SectorVn, details.Sector),
			Brief:       faFirstNonEmpty(faStripHTML(details.Profile), faStripHTML(details.EnProfile)),
			ShareholderStructure: map[string]any{
				"total_shares":           holders.TotalShares,
				"state_percentage":       holders.StatePercentage,
				"foreign_percentage":     holders.ForeignPercentage,
				"other_percentage":       holders.OtherPercentage,
				"board_percentage":       holders.BodPercentage,
				"institution_percentage": holders.InstitutionPercentage,
			},
			Ecosystem: faBuildEcosystem(rel),
		},
	}

	// Synthesis is non-fatal and never overwrites the factual fields above — only
	// the analytical narrative is merged in.
	if analysis, err := faSynthesizeAnalysis(ctx, s.agent, symbol, result.Overview, details); err != nil {
		slog.Warn("fundamental_analysis: synthesis failed", "symbol", symbol, "error", err)
	} else {
		result.BusinessActivity = analysis.BusinessActivity
		result.EconomicMoat = analysis.EconomicMoat
		result.Outlook = analysis.Outlook
		result.Risks = analysis.Risks
		result.Macro = analysis.Macro
	}

	common.WriteJSON(w, http.StatusOK, result)
}

// POST /v1/stock/research - Research tickers and return the whole result at once
func (s *Server) MarketResearchHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateRequest(r)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := s.stockDigestAgent(req, nil)
	go s.persistReports(response)
	common.WriteJSON(w, http.StatusOK, response)
}

// POST /v1/stock/research/stream - Research tickers, streaming progress as SSE
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
		doneCh <- s.stockDigestAgent(req, progressCh)
	}()

	for evt := range progressCh {
		common.WriteSSE(w, map[string]any{"type": "progress", "data": evt})

		if r.Context().Err() != nil {
			return
		}
	}

	response := <-doneCh
	common.WriteSSE(w, map[string]any{"type": "result", "data": response})

	go s.persistReports(response)
}

// ProcessResearchJob is the worker process function for research jobs.
func (s *Server) ProcessResearchJob(job *worker.ResearchJob) {
	ctx := context.Background()
	ticker := job.Ticker

	emit := func(step, message string, progress int) {
		if job.ProgressCh == nil {
			return
		}
		job.ProgressCh <- progressEvent{
			Ticker:   ticker,
			Step:     step,
			Message:  message,
			Progress: progress,
		}
	}

	// Wrapped in a closure so each failure can return early and still report the
	// result on the job's channel exactly once, below.
	run := func() (database.StockReport, error) {
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

		emit("submitting", "Submitting research request...", 25)
		requestID, err := s.services.Tavily.SubmitResearch(ctx, tavily.ResearchRequest{
			Input:        prompt,
			Model:        job.ResearchModel,
			Stream:       false,
			OutputSchema: tavily.ResearchOutputSchema(),
		})
		if err != nil {
			emit("failed", fmt.Sprintf("Failed to submit: %v", err), 0)
			return database.StockReport{}, fmt.Errorf("submit research for %s: %w", ticker, err)
		}
		slog.Info("tavily research submitted", "ticker", ticker, "request_id", requestID)

		emit("polling", "Gathering and analyzing data...", 40)
		pollResp, err := s.services.Tavily.PollResearch(ctx, requestID)
		if err != nil {
			emit("failed", fmt.Sprintf("Research timed out: %v", err), 0)
			return database.StockReport{}, fmt.Errorf("poll research for %s: %w", ticker, err)
		}

		emit("parsing", "Parsing research results...", 85)
		report, err := parseResearchResult(ticker, pollResp)
		if err != nil {
			emit("failed", fmt.Sprintf("Failed to parse: %v", err), 0)
			return database.StockReport{}, err
		}

		emit("completed", "Research complete", 100)
		return report, nil
	}

	report, err := run()
	job.ResultCh <- worker.ResearchResult{Ticker: ticker, Report: report, Err: err}
}

// GET /v1/stock/research-reports - List stored research reports with live prices
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

// GET /v1/stock/research-reports/{id} - Get one stored research report
func (s *Server) GetResearchReportByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	report, err := s.queries.GetResearchReportById(r.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, report)
}

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

func (s *Server) persistReports(report database.ResearchReport) {
	for _, ticker := range report.Reports {
		price, err := GetLatestMatchPrice(context.Background(), ticker.Ticker)
		if err != nil {
			slog.Error("Failed to fetch stock price", "ticker", ticker.Ticker, "error", err)
		}

		// The model returns prose like "BUY — strong Q2"; only the verdict is stored.
		recommendation := "Unknown"
		if words := strings.Fields(ticker.Recommendation); len(words) > 0 {
			recommendation = words[0]
		}

		_, err = s.queries.CreateResearchReport(context.Background(), database.CreateResearchReportParams{
			Ticker:         ticker.Ticker,
			Recommendation: recommendation,
			ReferencePrice: strconv.FormatFloat(price, 'f', -1, 64),
			Report:         ticker,
		})
		if err != nil {
			slog.Error("Failed to create research report", "ticker", ticker.Ticker, "error", err)
		}
	}
}

func (s *Server) stockDigestAgent(request ResearchRequestPayload, progressCh chan<- progressEvent) database.ResearchReport {
	resultCh := make(chan worker.ResearchResult, len(request.Tickers))

	// The worker pool speaks `chan any`; this proxy narrows it back to the typed
	// channel the SSE handler ranges over.
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

func parseResearchResult(ticker string, resp *tavily.ResearchPollResponse) (database.StockReport, error) {
	report := database.StockReport{Ticker: ticker}

	if len(resp.Content) == 0 {
		return report, nil
	}

	// Content arrives in three shapes and none of them is worth failing over:
	// structured JSON (Source has a custom UnmarshalJSON for string-or-object), a
	// plain string summary, or something unrecognised — which yields the ticker
	// alone rather than an error.
	if err := json.Unmarshal(resp.Content, &report); err != nil {
		var plainSummary string
		if err := json.Unmarshal(resp.Content, &plainSummary); err == nil {
			report.Summary = plainSummary
			return report, nil
		}
		return report, nil
	}

	// The model may omit the ticker or change its casing.
	report.Ticker = ticker

	// Tavily's own sources carry titles, so they fill in when the model gave none.
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
