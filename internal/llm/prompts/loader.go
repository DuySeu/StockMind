package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
	"time"
)

//go:embed templates/*.txt
var templatesFS embed.FS

// templates is parsed once at startup — any syntax error in .txt files
// will cause a panic immediately, rather than a silent runtime failure.
var templates = template.Must(
	template.ParseFS(templatesFS, "templates/*.txt"),
)

// ResearchParams holds the variables for the research prompt.
type ResearchParams struct {
	Ticker string
	Date   string
}

// MetricsParams holds the variables for the metrics prompt.
type MetricsParams struct {
	Ticker  string
	Content string
}

// SummarizationParams holds the variables for the summarization prompt.
type SummarizationParams struct {
	Summary  string
	KeyFacts string
	Messages string
}

// SystemParams holds the variables for the system prompt.
type SystemParams struct {
	Date     string
	Summary  string
	KeyFacts string
}

// FundamentalAnalysisParams holds the variables for the fundamental analysis prompt.
type FundamentalAnalysisParams struct {
	Symbol string
	Facts  string
}

// PlanParams holds the variables for the multi-agent planning prompt.
// Roster is the pre-rendered catalogue of available agents; Errors carries
// validation failures on a repair retry and is empty on the first attempt.
type PlanParams struct {
	Goal   string
	Roster string
	Date   string
	Errors string
}

// AgentPromptParams holds the variables shared by every specialist agent's role
// prompt. Date is injected automatically so agents can reason about "today".
type AgentPromptParams struct {
	Date string
}

// PromptLoader renders prompts from embedded Go templates.
type PromptLoader struct{}

// NewPromptLoader creates a new prompt loader.
func NewPromptLoader() *PromptLoader {
	return &PromptLoader{}
}

// render executes a named template with the given data and returns the result.
func (p *PromptLoader) render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render prompt %s: %w", name, err)
	}
	return buf.String(), nil
}

// GetResearchPrompt renders the research prompt with the given params.
func (p *PromptLoader) GetResearchPrompt(params ResearchParams) (string, error) {
	return p.render("research_prompt.txt", params)
}

// GetMetricsPrompt renders the metrics prompt with the given params.
func (p *PromptLoader) GetMetricsPrompt(params MetricsParams) (string, error) {
	return p.render("metrics_prompt.txt", params)
}

// GetSummarizationPrompt renders the summarization prompt with the given params.
func (p *PromptLoader) GetSummarizationPrompt(params SummarizationParams) (string, error) {
	return p.render("summarization_prompt.txt", params)
}

// GetSummaryPrompt renders the summary prompt with the given params.
func (p *PromptLoader) GetSystemPrompt(params SystemParams) (string, error) {
	params.Date = time.Now().Format("2006-01-02")
	return p.render("system_prompt.txt", params)
}

// GetFundamentalAnalysisPrompt renders the fundamental analysis prompt with the given params.
func (p *PromptLoader) GetFundamentalAnalysisPrompt(params FundamentalAnalysisParams) (string, error) {
	return p.render("fundamental_analysis.txt", params)
}

// GetPlanPrompt renders the multi-agent planning prompt with the given params.
func (p *PromptLoader) GetPlanPrompt(params PlanParams) (string, error) {
	params.Date = time.Now().Format("2006-01-02")
	return p.render("plan_prompt.txt", params)
}

// GetAgentPrompt renders a specialist agent's role prompt by template file name
// (e.g. "agent_market_data.txt"). Generic on purpose: adding an agent needs a new
// .txt file but no new loader method.
func (p *PromptLoader) GetAgentPrompt(tplName string) (string, error) {
	return p.render(tplName, AgentPromptParams{Date: time.Now().Format("2006-01-02")})
}
