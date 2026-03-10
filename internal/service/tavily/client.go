package tavily

import (
	"net/http"
	"os"
	"stockmind/internal/common"
	"time"
)

const defaultHTTPTimeout = 30 * time.Second

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewClient creates a Tavily API client with default settings.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		baseURL:    common.TAVILY_URL,
		apiKey:     os.Getenv("TAVILY_API_KEY"),
	}
}
