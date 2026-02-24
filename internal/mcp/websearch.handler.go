package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"stockmind/internal/common"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
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

func WebSearchResult(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Prepare request with native domain filtering
	// Use configured search depth, default to "basic" if not set
	searchDepth := "basic" // "basic" or "advanced"

	reqBody := tavilyRequest{
		APIKey:            os.Getenv("TAVILY_API_KEY"),
		Query:             query,
		SearchDepth:       searchDepth,
		IncludeAnswer:     true,
		IncludeImages:     false,
		IncludeRawContent: false,
		MaxResults:        10,
		IncludeDomains:    []string{"vnexpress.net"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	http_req, err := http.NewRequestWithContext(ctx, "POST", common.TAVILY_URL+"/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	http_req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Execute request
	resp, err := client.Do(http_req)
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

	return mcp.NewToolResultStructuredOnly(tavilyResp.Results), nil
}
