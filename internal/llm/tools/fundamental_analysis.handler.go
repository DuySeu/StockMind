package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"stockmind/internal/common"
)

// ──────── Tool I/O types ────────

type FundamentalAnalysisInput struct {
	Symbol string `json:"symbol" jsonschema:"Stock symbol, e.g., HPG"`
}

func (FundamentalAnalysisInput) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"symbol": map[string]any{
				"type":        "string",
				"description": "Stock symbol, e.g., HPG",
			},
		},
		"required": []string{"symbol"},
	}
}

type Output struct {
	Overview      Overview       `json:"overview"`
	ExtraAnalysis companyDetails `json:"extra_analysis"`
	Symbol        string         `json:"symbol"`
}

type Overview struct {
	CompanyName          string         `json:"company_name"`
	Industry             string         `json:"industry"`
	Brief                string         `json:"brief"`
	ShareholderStructure map[string]any `json:"shareholder_structure"`
	Ecosystem            map[string]any `json:"ecosystem,omitempty"`
}

// ──────── VietCap iq-insight-service response shapes ────────

type companyDetails struct {
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

type shareholderStructure struct {
	TotalShares           float64 `json:"totalShares"`
	StatePercentage       float64 `json:"statePercentage"`
	ForeignPercentage     float64 `json:"foreignPercentage"`
	OtherPercentage       float64 `json:"otherPercentage"`
	BodPercentage         float64 `json:"bodPercentage"`
	InstitutionPercentage float64 `json:"institutionPercentage"`
}

type relatedOrg struct {
	RightOrganCode   string  `json:"rightOrganCode"`
	RightTicker      string  `json:"rightTicker"`
	RightOrganNameVi string  `json:"rightOrganNameVi"`
	RightOrganNameEn string  `json:"rightOrganNameEn"`
	OwnedPercentage  float64 `json:"ownedPercentage"`
}

type relationship struct {
	Affiliates   []relatedOrg `json:"affiliates"`
	Subsidiaries []relatedOrg `json:"subsidiaries"`
}

// ──────── Handler ────────

// HandleFundamentalAnalysis builds a hybrid fundamental analysis: factual fields
// (overview, shareholder structure, ecosystem) are fetched from VietCap's
// iq-insight-service and kept authoritative, while the analytical narrative
// (business activity, economic moat, outlook, risks, macro) is synthesized by
// the LLM, grounded on that factual data.
func HandleFundamentalAnalysis(ctx context.Context, input FundamentalAnalysisInput) (any, error) {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// 1. Company details (required for grounding).
	details, err := fetchIQ[companyDetails](ctx, fmt.Sprintf("%s/details?ticker=%s", common.COMPANY_URL, symbol))
	if err != nil {
		return nil, fmt.Errorf("fetch company details for %s: %w", symbol, err)
	}
	if details.ViOrganName == "" && details.EnOrganName == "" {
		return nil, fmt.Errorf("no company data found for %s", symbol)
	}

	// 2. Shareholders + relationships (best-effort, fetched concurrently).
	var (
		wg      sync.WaitGroup
		holders shareholderStructure
		rel     relationship
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		h, err := fetchIQ[shareholderStructure](ctx, fmt.Sprintf("%s/%s/shareholder-structure", common.COMPANY_URL, symbol))
		if err != nil {
			log.Printf("fundamental_analysis: shareholder structure for %s: %v", symbol, err)
			return
		}
		holders = h
	}()
	go func() {
		defer wg.Done()
		r, err := fetchIQ[relationship](ctx, fmt.Sprintf("%s/%s/relationship", common.COMPANY_URL, symbol))
		if err != nil {
			log.Printf("fundamental_analysis: relationship for %s: %v", symbol, err)
			return
		}
		rel = r
	}()
	wg.Wait()

	// 3. Build authoritative factual fields.
	companyName := firstNonEmpty(details.ViOrganName, details.EnOrganName)
	industry := firstNonEmpty(details.SectorVn, details.Sector)
	brief := firstNonEmpty(stripHTML(details.Profile), stripHTML(details.EnProfile))

	output := Output{
		Overview: Overview{
			CompanyName: companyName,
			Industry:    industry,
			Brief:       brief,
			ShareholderStructure: map[string]any{
				"total_shares":           holders.TotalShares,
				"state_percentage":       holders.StatePercentage,
				"foreign_percentage":     holders.ForeignPercentage,
				"other_percentage":       holders.OtherPercentage,
				"board_percentage":       holders.BodPercentage,
				"institution_percentage": holders.InstitutionPercentage,
			},
			Ecosystem: buildEcosystem(rel),
		},
		ExtraAnalysis: details,
		Symbol:        symbol,
	}

	return output, nil
}

// ──────── Helpers ────────

// fetchIQ performs a GET against the VietCap iq-insight-service, decompresses the
// response, and unwraps the standard {status, data} envelope into T.
func fetchIQ[T any](ctx context.Context, path string) (T, error) {
	var env struct {
		Successful bool `json:"successful"`
		Data       T    `json:"data"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, common.IQ_INSIGHT_URL+path, nil)
	if err != nil {
		return env.Data, err
	}
	for k, v := range common.VCI_HEADERS {
		req.Header.Set(k, v)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return env.Data, err
	}
	defer resp.Body.Close()

	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return env.Data, err
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return env.Data, err
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return env.Data, fmt.Errorf("unmarshal iq-insight response: %w", err)
	}
	return env.Data, nil
}

// buildEcosystem maps subsidiaries/affiliates into a serializable structure.
// Returns nil when no relationships exist (field is omitempty).
func buildEcosystem(rel relationship) map[string]any {
	if len(rel.Subsidiaries) == 0 && len(rel.Affiliates) == 0 {
		return nil
	}
	toList := func(orgs []relatedOrg) []map[string]any {
		out := make([]map[string]any, 0, len(orgs))
		for _, o := range orgs {
			out = append(out, map[string]any{
				"name":             firstNonEmpty(o.RightOrganNameVi, o.RightOrganNameEn),
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

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags, decodes entities, and collapses whitespace.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
