package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// tavilyRequest represents the Tavily API request
type tavilyRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth"`
	IncludeAnswer     bool     `json:"include_answer"`
	IncludeImages     bool     `json:"include_images"`
	IncludeRawContent bool     `json:"include_raw_content"`
	MaxResults        int      `json:"max_results"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
}

// tavilyResponse represents the Tavily API response
type tavilyResponse struct {
	Success bool   `json:"success"`
	Answer  string `json:"answer"`
	Results []struct {
		Title      string  `json:"title"`
		URL        string  `json:"url"`
		Content    string  `json:"content"`
		RawContent string  `json:"raw_content"`
		Score      float64 `json:"score"`
	} `json:"results"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// SubmitResearch kicks off an async research job and returns the request ID.
func (c *Client) SearchWeb(ctx context.Context, query string, includeDomains []string) ([]SearchResult, error) {
	reqBody := tavilyRequest{
		APIKey:            c.apiKey,
		Query:             query,
		SearchDepth:       "basic",
		IncludeAnswer:     true,
		IncludeImages:     false,
		IncludeRawContent: false,
		MaxResults:        5,
		IncludeDomains:    includeDomains,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily API returned status %d", resp.StatusCode)
	}

	// Parse response
	var tavilyResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we have results, regardless of Success field
	// Tavily might return Success=false but still have results
	if len(tavilyResp.Results) == 0 {
		return nil, fmt.Errorf("tavily search returned no results")
	}

	// Convert to SearchResult - prefer raw_content for full text
	results := make([]SearchResult, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		// Use raw_content if available (full page content), otherwise use content (snippet)
		description := r.Content
		if r.RawContent != "" && len(r.RawContent) > len(r.Content) {
			description = r.RawContent
		}

		results = append(results, SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Description: description,
		})
	}
	return results, nil
}
