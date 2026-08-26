package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"
)

var faHTMLTagRe = regexp.MustCompile(`<[^>]*>`)

// faSynthesis holds only the analytical fields synthesized by the LLM. The
// factual ones are fetched, never generated.
type faSynthesis struct {
	BusinessActivity BusinessActivity `json:"business_activity"`
	EconomicMoat     string           `json:"economic_moat"`
	Outlook          Outlook          `json:"outlook"`
	Risks            Risks            `json:"risks"`
	Macro            string           `json:"macro"`
}

// The fa* shapes below mirror VietCap's iq-insight-service responses; the field
// tags are its camelCase names, not ours.
type faCompanyDetails struct {
	Ticker              string  `json:"ticker"`
	ViOrganName         string  `json:"viOrganName"`
	ViOrganShortName    string  `json:"viOrganShortName"`
	EnOrganName         string  `json:"enOrganName"`
	Sector              string  `json:"sector"`
	SectorVn            string  `json:"sectorVn"`
	Profile             string  `json:"profile"`
	EnProfile           string  `json:"enProfile"`
	MarketCap           float64 `json:"marketCap"`
	CurrentPrice        float64 `json:"currentPrice"`
	Rating              string  `json:"rating"`
	TargetPrice         float64 `json:"targetPrice"`
	ForeignerPercentage float64 `json:"foreignerPercentage"`
}

type faShareholderStructure struct {
	TotalShares           float64 `json:"totalShares"`
	StatePercentage       float64 `json:"statePercentage"`
	ForeignPercentage     float64 `json:"foreignPercentage"`
	OtherPercentage       float64 `json:"otherPercentage"`
	BodPercentage         float64 `json:"bodPercentage"`
	InstitutionPercentage float64 `json:"institutionPercentage"`
}

type faRelatedOrg struct {
	RightOrganCode   string  `json:"rightOrganCode"`
	RightTicker      string  `json:"rightTicker"`
	RightOrganNameVi string  `json:"rightOrganNameVi"`
	RightOrganNameEn string  `json:"rightOrganNameEn"`
	OwnedPercentage  float64 `json:"ownedPercentage"`
}

type faRelationship struct {
	Affiliates   []faRelatedOrg `json:"affiliates"`
	Subsidiaries []faRelatedOrg `json:"subsidiaries"`
}

// faSynthesizeAnalysis prompts the LLM to produce the analytical narrative
// fields, grounded on the supplied factual data.
func faSynthesizeAnalysis(ctx context.Context, agent *core.LLMService, symbol string, overview Overview, details faCompanyDetails) (faSynthesis, error) {
	var result faSynthesis
	facts, _ := json.MarshalIndent(map[string]any{
		"symbol":         symbol,
		"company_name":   overview.CompanyName,
		"industry":       overview.Industry,
		"profile":        overview.Brief,
		"shareholders":   overview.ShareholderStructure,
		"ecosystem":      overview.Ecosystem,
		"market_cap":     details.MarketCap,
		"analyst_rating": details.Rating,
		"target_price":   details.TargetPrice,
	}, "", "  ")

	loader := prompts.NewPromptLoader()
	prompt, err := loader.GetFundamentalAnalysisPrompt(prompts.FundamentalAnalysisParams{
		Symbol: symbol,
		Facts:  string(facts),
	})
	if err != nil {
		return result, fmt.Errorf("build fundamental analysis prompt: %w", err)
	}

	if err := agent.StructuredCompletion(ctx, prompt, &result); err != nil {
		return result, err
	}
	return result, nil
}

// ──────── Helpers ────────

// faBuildEcosystem maps subsidiaries/affiliates into a serializable structure.
// Returns nil when no relationships exist (field is omitempty).
func faBuildEcosystem(rel faRelationship) map[string]any {
	if len(rel.Subsidiaries) == 0 && len(rel.Affiliates) == 0 {
		return nil
	}
	toList := func(orgs []faRelatedOrg) []map[string]any {
		out := make([]map[string]any, 0, len(orgs))
		for _, o := range orgs {
			out = append(out, map[string]any{
				"name":             faFirstNonEmpty(o.RightOrganNameVi, o.RightOrganNameEn),
				"ticker":           o.RightTicker,
				"owned_percentage": o.OwnedPercentage,
			})
		}
		return out
	}
	return map[string]any{
		"subsidiaries": toList(rel.Subsidiaries),
		"affiliates":   toList(rel.Affiliates),
	}
}

// faStripHTML removes HTML tags, decodes entities, and collapses whitespace:
// the VietCap profile fields arrive as HTML fragments.
func faStripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = faHTMLTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func faFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
