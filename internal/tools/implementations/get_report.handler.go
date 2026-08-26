package implementations

import (
	"context"
	"fmt"
	"slices"

	"stockmind/internal/common"
)

const (
	// statistics-financial tags every row with the basis it was computed on:
	// RATIO_TTM rows are trailing-twelve-month figures stamped with the quarter
	// they end in, RATIO_YEAR rows are annual. The tag is authoritative; the
	// quarter number is not (annual rows carry quarter 5).
	ratioBasisTTM    = "RATIO_TTM"
	ratioBasisAnnual = "RATIO_YEAR"

	defaultReportPeriods = 8
	maxReportPeriods     = 20
)

// reportInternalFields are VietCap's bookkeeping keys. They carry no analytical
// meaning and only cost model context.
var reportInternalFields = []string{"ratioTTMId", "ratioYearId", "ratioType", "organCode"}

type GetReportInput struct {
	Symbol string `json:"symbol"`
	Period string `json:"period,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// Schema declares the tool's input contract, including the period enum.
func (GetReportInput) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"symbol": map[string]any{
				"type":        "string",
				"description": "Stock symbol, e.g., HPG",
			},
			"period": map[string]any{
				"type":        "string",
				"description": "Q for trailing-twelve-month ratios stamped by quarter, Y for annual ratios",
				"enum":        []string{"Q", "Y"},
				"default":     "Q",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many periods to return, newest first (default %d, max %d)", defaultReportPeriods, maxReportPeriods),
				"default":     defaultReportPeriods,
			},
		},
		"required": []string{"symbol"},
	}
}

// HandleGetReport fetches a symbol's financial ratios for the requested basis,
// newest period first.
func HandleGetReport(ctx context.Context, input GetReportInput) (any, error) {
	if input.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// Resolve the basis and how many periods to return
	basis := ratioBasisTTM
	if input.Period == "Y" {
		basis = ratioBasisAnnual
	}
	limit := input.Limit
	if limit <= 0 {
		limit = defaultReportPeriods
	}
	limit = min(limit, maxReportPeriods)

	rows, err := common.FetchIQInsight[[]map[string]any](ctx,
		fmt.Sprintf("%s/%s/statistics-financial", common.COMPANY_URL, input.Symbol))
	if err != nil {
		return nil, fmt.Errorf("fetch ratios for %s: %w", input.Symbol, err)
	}

	selected := selectRatioRows(rows, basis, limit)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no %s ratios found for %s", basis, input.Symbol)
	}

	label := "trailing_twelve_months"
	if basis == ratioBasisAnnual {
		label = "annual"
	}

	return map[string]any{
		"symbol":  input.Symbol,
		"basis":   label,
		"periods": pruneRatioRows(selected),
	}, nil
}

// selectRatioRows keeps the rows matching the requested basis and returns at
// most limit of them, newest first.
// style: keep — the ordering flip is the one thing here that has to be tested
// without a network call, since VietCap returns rows oldest-first.
func selectRatioRows(rows []map[string]any, basis string, limit int) []map[string]any {
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if tag, _ := row["ratioType"].(string); tag == basis {
			selected = append(selected, row)
		}
	}

	// VietCap returns oldest first; the newest period matters most to the model
	slices.Reverse(selected)

	if len(selected) > limit {
		selected = selected[:limit]
	}

	return selected
}

// pruneRatioRows drops VietCap's bookkeeping keys and every metric that is
// absent or zero across all of the selected rows.
//
// The pruning is dynamic rather than a fixed field list on purpose: the ratio
// set differs by company type, so a list tuned on a corporate would silently
// discard a bank's nim, npl, car and casaRatio.
func pruneRatioRows(rows []map[string]any) []map[string]any {
	// Find every key that carries a real value in at least one row
	meaningful := make(map[string]bool, len(rows[0]))
	for _, row := range rows {
		for key, value := range row {
			if slices.Contains(reportInternalFields, key) {
				continue
			}
			if number, isNumber := value.(float64); isNumber && number == 0 {
				continue
			}
			if value == nil {
				continue
			}
			meaningful[key] = true
		}
	}

	pruned := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		kept := make(map[string]any, len(meaningful))
		for key, value := range row {
			if meaningful[key] {
				kept[key] = value
			}
		}
		pruned = append(pruned, kept)
	}

	return pruned
}
