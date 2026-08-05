package database

import (
	"encoding/json"
)

type StreamEventType string
type ModelProvider string

const (
	EventThinking           StreamEventType = "thinking"
	EventText               StreamEventType = "text"
	EventToolCall           StreamEventType = "tool_call"
	EventToolResult         StreamEventType = "tool_result"
	EventError              StreamEventType = "error"
	EventDone               StreamEventType = "done"
	ModelProviderAnthropic  ModelProvider   = "anthropic"
	ModelProviderAWS        ModelProvider   = "aws"
	ModelProviderOpenAI     ModelProvider   = "openai"
	ModelProviderOpenRouter ModelProvider   = "openrouter"
)

type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Content string          `json:"content,omitempty"` // For text delta
	Data    any             `json:"data,omitempty"`    // For Error or ToolCall/Result details
}

// ToolConfig represents a tool definition stored in agent flow JSONB configs.
type ToolConfig struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

type AgentConfig struct {
	Description   string        `json:"description"`
	SystemPrompt  string        `json:"systemPrompt"`
	Provider      ModelProvider `json:"provider"` // anthropic or openai
	ModelID       string        `json:"modelId"`
	MaxTokens     int64         `json:"maxTokens"`
	Temperature   float64       `json:"temperature"`
	TopP          float64       `json:"topP"`
	TopK          int64         `json:"topK"`
	ThinkingToken int64         `json:"thinkingToken"`
	Tools         []ToolConfig  `json:"tools"`
	McpServers    []MCPConfig   `json:"mcpServers"` // MCP servers to use
}

type Metadata struct {
	Tool        []Tool       `json:"tools,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Sources     []Source     `json:"sources,omitempty"`
	Error       *TurnError   `json:"error,omitempty"`
	// Thinking is the reasoning shown alongside a turn — for a max-mode turn, the
	// plan and each step's prose. Stored so a reload shows the same turn the stream
	// showed instead of dropping the explanation and keeping only the conclusion.
	Thinking string `json:"thinking,omitempty"`
}

// TurnError records why an assistant turn failed. Persisting it is what keeps a
// failed turn from reading as a silent gap after a reload: without it the user's
// question was stored and the reply simply never appeared.
//
// Message is client-safe prose — the underlying cause is logged server-side, not
// sent — and Code is a stable identifier the UI can branch on.
type TurnError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type Tool struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   string `json:"is_error,omitempty"`
}

type Attachment struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Path      string `json:"path"`
	Data      []byte `json:"-"` // transient: resolved at runtime, not persisted
}

type MCPConfig struct {
	Name           string            `json:"name"`
	Protocol       string            `json:"protocol"` // stdio, streamablehttp
	Command        *string           `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Envs           map[string]string `json:"envs,omitempty"`
	URL            *string           `json:"url"` // Remote MCP server URL
	Authentication *string           `json:"key"` // API key or token for authentication
}

type AgentFlowConfig struct {
	Agents map[string]AgentConfig `json:"agents"`
}

// ResearchReport is the outbound payload for the research endpoint.
type ResearchReport struct {
	Reports     map[string]StockReport `json:"reports"`
	GeneratedAt string                 `json:"generated_at"`
}

// StockReport holds the analysis returned for a single ticker.
type StockReport struct {
	Ticker             string   `json:"ticker"`
	CompanyName        string   `json:"company_name"`
	Summary            string   `json:"summary"`
	CurrentPerformance string   `json:"current_performance"`
	KeyInsights        []string `json:"key_insights"`
	Recommendation     string   `json:"recommendation"` // Buy/hold/sell with reasoning
	RiskAssessment     string   `json:"risk_assessment"`
	PriceOutlook       string   `json:"price_outlook"`
	MarketCap          string   `json:"market_cap"`
	PERatio            string   `json:"pe_ratio"`
	Sources            []Source `json:"sources"`
}

// Source is a reference URL used in the research.
type Source struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ConversationSummary is stored in conversations.metadata JSONB.
type ConversationSummary struct {
	Summary         string   `json:"summary"`
	KeyFacts        []string `json:"key_facts"`
	SummarizedCount int64    `json:"summarized_count"`
}

// UnmarshalJSON allows Source to be deserialized from either a plain string
// (just a URL) or a JSON object with "url" and "title" fields.
func (s *Source) UnmarshalJSON(data []byte) error {
	// Try plain string first (e.g. "https://example.com").
	var url string
	if err := json.Unmarshal(data, &url); err == nil {
		s.URL = url
		return nil
	}

	// Otherwise, parse as a structured object.
	type sourceAlias Source // avoid infinite recursion
	var alias sourceAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*s = Source(alias)
	return nil
}
