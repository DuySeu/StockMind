package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stockmind/internal/common"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type PiotroskiEvaluation struct {
	Symbol  string  `json:"symbol"`
	Period  string  `json:"period"`
	Score   int     `json:"score"`
	Details Details `json:"details"`
}

type Details struct {
	NetIncome              bool `json:"net_income"`
	ROA                    bool `json:"roa"`
	NetOperatingCashFlow   bool `json:"net_operating_cash_flow"`
	CashFlowFromOperations bool `json:"cash_flow_from_operations"`
	LongTermDebt           bool `json:"long_term_debt"`
	CurrentRatio           bool `json:"current_ratio"`
	NewsIssued             bool `json:"news_issued"`
	GrossMargin            bool `json:"gross_margin"`
	AssetTurnoverRatio     bool `json:"asset_turnover_ratio"`
}

const piotroskiQuery = "fragment Ratios on CompanyFinancialRatio { ticker yearReport revenue revenueGrowth netProfit roa currentRatio grossMargin at issueShare pe eps pcf de le __typename } query Query($ticker: String!, $period: String!) { CompanyFinancialRatio(ticker: $ticker, period: $period) { ratio { ...Ratios __typename } period __typename } }"

func GetPiotroskiEvaluation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol, err := request.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Create payload struct
	payload := GraphQLPayload{
		Query: piotroskiQuery,
		Variables: map[string]interface{}{
			"ticker": symbol,
			"period": "Q",
		},
	}

	// Marshal to JSON bytes
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	http_req, err := http.NewRequestWithContext(ctx, "POST", common.GRAPHQL_URL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Write headers from VCI_HEADERS
	for k, v := range common.VCI_HEADERS {
		http_req.Header.Set(k, v)
	}

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(http_req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer resp.Body.Close()

	// Check for GZIP compression
	reader, err := common.GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer reader.Close()

	// Read response
	body, err := io.ReadAll(reader)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Unmarshal response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract data
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid response format: data field missing or not a map"), nil
	}
	financialRatio, ok := data["CompanyFinancialRatio"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid response format: CompanyFinancialRatio missing or not a map"), nil
	}
	ratios, ok := financialRatio["ratio"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid response format: ratio missing or not a list"), nil
	}
	periods, ok := financialRatio["period"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid response format: period missing or not a list"), nil
	}

	if len(periods) == 0 || len(ratios) == 0 {
		return mcp.NewToolResultError("no financial data found"), nil
	}

	latestPeriod := periods[0].(string)

	if len(ratios) < 5 {
		return mcp.NewToolResultError("insufficient data for year-over-year comparison (need at least 5 quarters)"), nil
	}

	current := ratios[0].(map[string]interface{})
	prevYear := ratios[4].(map[string]interface{}) // approximate same quarter last year

	// Helper to safely get float64
	getFloat := func(m map[string]interface{}, key string) float64 {
		if val, ok := m[key]; ok && val != nil {
			return val.(float64)
		}
		return 0.0
	}

	// 1. Net Income
	netIncome := getFloat(current, "netProfit")
	scoreNetIncome := netIncome > 0

	// 2. ROA
	roa := getFloat(current, "roa")
	scoreROA := roa > 0

	// 3. Operating Cash Flow (Derived)
	// OCF = MarketCap / PCF = (PE * EPS * Shares) / PCF
	pe := getFloat(current, "pe")
	eps := getFloat(current, "eps")
	shares := getFloat(current, "issueShare")
	pcf := getFloat(current, "pcf")

	var ocf float64
	var scoreOCF bool
	if pcf != 0 {
		price := pe * eps
		marketCap := price * shares
		ocf = marketCap / pcf
		scoreOCF = ocf > 0
	}

	// 3. Cash Flow from Operations
	scoreCFO := ocf > netIncome

	// 4. Quality of Earnings (OCF > Net Income)
	scoreQualityOfEarnings := ocf > netIncome

	// 5. Long Term Debt (Derived from DE)
	// D/E = Debt/Equity => Debt/Assets = (D/E) / (1 + D/E)
	deCurr := getFloat(current, "de")
	dePrev := getFloat(prevYear, "de")
	levCurr := 0.0
	if deCurr != -1 { // Assuming -1 or check logic, but usually > 0
		levCurr = deCurr / (1 + deCurr)
	}
	levPrev := 0.0
	if dePrev != -1 {
		levPrev = dePrev / (1 + dePrev)
	}
	scoreLTD := levCurr < levPrev

	// 6. Current Ratio
	crCurr := getFloat(current, "currentRatio")
	crPrev := getFloat(prevYear, "currentRatio")
	scoreCR := crCurr > crPrev

	// 7. Dilution (Shares Issued)
	sharesPrev := getFloat(prevYear, "issueShare")
	scoreDilution := shares <= sharesPrev

	// 8. Gross Margin
	gmCurr := getFloat(current, "grossMargin")
	gmPrev := getFloat(prevYear, "grossMargin")
	scoreGM := gmCurr > gmPrev

	// 9. Asset Turnover
	atCurr := getFloat(current, "at")
	atPrev := getFloat(prevYear, "at")
	scoreAT := atCurr > atPrev

	// Calculate Total Score
	score := 0
	if scoreNetIncome {
		score++
	}
	if scoreROA {
		score++
	}
	if scoreOCF {
		score++
	}
	if scoreQualityOfEarnings {
		score++
	}
	if scoreLTD {
		score++
	}
	if scoreCR {
		score++
	}
	if scoreDilution {
		score++
	}
	if scoreGM {
		score++
	}
	if scoreAT {
		score++
	}

	evaluation := PiotroskiEvaluation{
		Symbol: symbol,
		Period: latestPeriod,
		Score:  score,
		Details: Details{
			NetIncome:              scoreNetIncome,
			ROA:                    scoreROA,
			NetOperatingCashFlow:   scoreOCF,
			CashFlowFromOperations: scoreCFO,
			LongTermDebt:           scoreLTD,
			CurrentRatio:           scoreCR,
			NewsIssued:             scoreDilution,
			GrossMargin:            scoreGM,
			AssetTurnoverRatio:     scoreAT,
		},
	}
	return mcp.NewToolResultStructuredOnly(evaluation), nil
}
