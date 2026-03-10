package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"stockmind/internal/database"
	"time"
)

const (
	researchPollInterval = 2 * time.Second
	researchPollTimeout  = 2 * time.Minute
)

// ResearchRequest is the POST body sent to Tavily /research.
type ResearchRequest struct {
	Input        string      `json:"input"`
	Model        string      `json:"model"`
	Stream       bool        `json:"stream"`
	OutputSchema interface{} `json:"output_schema,omitempty"`
}

// researchInitResponse is the initial async response (unexported).
type researchInitResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
}

// ResearchPollResponse is returned when polling a research job.
type ResearchPollResponse struct {
	Status  string            `json:"status"`
	Content json.RawMessage   `json:"content"`
	Sources []database.Source `json:"sources"`
}

// SubmitResearch kicks off an async research job and returns the request ID.
func (c *Client) SubmitResearch(ctx context.Context, req ResearchRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/research", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create POST request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("tavily POST /research: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("tavily POST /research returned %d: %s", res.StatusCode, string(b))
	}

	var initResp researchInitResponse
	if err := json.NewDecoder(res.Body).Decode(&initResp); err != nil {
		return "", fmt.Errorf("decode tavily init response: %w", err)
	}

	if initResp.RequestID == "" {
		return "", fmt.Errorf("tavily returned empty request_id (status=%s)", initResp.Status)
	}

	return initResp.RequestID, nil
}

// PollResearch polls Tavily until the research is "completed", "failed" or "cancelled".
func (c *Client) PollResearch(ctx context.Context, requestID string) (*ResearchPollResponse, error) {
	ticker := time.NewTicker(researchPollInterval)
	defer ticker.Stop()

	deadline := time.After(researchPollTimeout)

	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("tavily poll timed out after %v", researchPollTimeout)
		case <-ticker.C:
			// continue polling below
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/research/"+requestID, nil)
		if err != nil {
			slog.Warn("tavily poll request error, retrying", "attempt", attempt, "error", err)
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		res, err := c.httpClient.Do(httpReq)
		if err != nil {
			slog.Warn("tavily poll error, retrying", "attempt", attempt, "error", err)
			continue
		}

		var pollResp ResearchPollResponse
		decErr := json.NewDecoder(res.Body).Decode(&pollResp)
		res.Body.Close()
		if decErr != nil {
			slog.Warn("tavily poll decode error, retrying", "attempt", attempt, "error", decErr)
			continue
		}

		slog.Info("tavily poll", "attempt", attempt, "status", pollResp.Status, "request_id", requestID)

		switch pollResp.Status {
		case "completed":
			return &pollResp, nil
		case "failed", "error":
			return nil, fmt.Errorf("tavily research failed (request_id=%s)", requestID)
		}
	}
}

// ResearchOutputSchema returns the schema definition for structured research output.
func ResearchOutputSchema() map[string]interface{} {
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
