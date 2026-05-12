package database

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"
)

type StopReason string

const (
	StopReasonMaxTokens  StopReason = "max_tokens"
	StopReasonUserInput  StopReason = "user_input"
	StopReasonToolCall   StopReason = "tool_call"
	StopReasonToolResult StopReason = "tool_result"
	StopReasonAgentDone  StopReason = "agent_done"
	StopReasonUnknown    StopReason = "unknown"
	StopReasonNil        StopReason = ""
)

type Node struct {
	ID        string      `json:"id"`
	Type      NodeType    `json:"type"` // start, agent
	AgentName *string     `json:"agentName,omitempty"`
	Next      *string     `json:"next,omitempty"`
	Output    *NodeOutput `json:"output,omitempty"`
}

type NodeType string
type NodeOutputType string
type NodeContentRole string
type ModelProvider string

const (
	NodeTypeStart            NodeType        = "start"
	NodeTypeAgent            NodeType        = "agent"
	NodeOutputTypeText       NodeOutputType  = "text"
	NodeOutputTypeStructured NodeOutputType  = "structured"
	NodeContentRoleUser      NodeContentRole = "user"
	NodeContentRoleSystem    NodeContentRole = "system"
	ModelProviderAnthropic   ModelProvider   = "anthropic"
	ModelProviderAWS         ModelProvider   = "aws"
	ModelProviderOpenAI      ModelProvider   = "openai"
	ModelProviderOpenRouter  ModelProvider   = "openrouter"
)

type NodeOutput struct {
	Type          NodeOutputType  `json:"type"` // text or structured (JSON)
	ContentFormat string          `json:"contentFormat"`
	ContentRole   NodeContentRole `json:"contentRole"` // user, system, assistant
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
	Tools         []mcp.Tool    `json:"tools"`
	McpServers    []MCPConfig   `json:"mcpServers"` // MCP servers to use
}

type Metadata struct {
	Tool        []Tool       `json:"toolCalls"`
	Attachments []Attachment `json:"attachments"`
	Sources     []Source     `json:"sources"`
}

type Tool struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Output    string `json:"output"`
	IsError   string `json:"is_error"`
}

type Attachment struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Data      []byte `json:"data"`
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
	Nodes  []Node                 `json:"nodes"`
}

type MessageUnion struct {
	OfOpenAI    *openai.ChatCompletionMessage `json:"of_openai,omitempty"`
	OfAnthropic *anthropic.MessageParam       `json:"of_anthropic,omitempty"`
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
