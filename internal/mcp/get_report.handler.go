package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// GraphQLPayload represents the GraphQL request payload
type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

const query = "fragment Ratios on CompanyFinancialRatio { ticker yearReport lengthReport updateDate revenue revenueGrowth netProfit netProfitGrowth ebitMargin roe roic roa pe pb eps currentRatio cashRatio quickRatio interestCoverage ae netProfitMargin grossMargin ev issueShare ps pcf bvps evPerEbitda BSA1 BSA2 BSA5 BSA8 BSA10 BSA159 BSA16 BSA22 BSA23 BSA24 BSA162 BSA27 BSA29 BSA43 BSA46 BSA50 BSA209 BSA53 BSA54 BSA55 BSA56 BSA58 BSA67 BSA71 BSA173 BSA78 BSA79 BSA80 BSA175 BSA86 BSA90 BSA96 CFA21 CFA22 at fat acp dso dpo ccc de le ebitda ebit dividend RTQ10 charterCapitalRatio RTQ4 epsTTM charterCapital fae RTQ17 CFA26 CFA6 CFA9 BSA85 CFA36 BSB98 BSB101 BSA89 CFA34 CFA14 ISB34 ISB27 ISA23 ISS152 ISA102 CFA27 CFA12 CFA28 BSA18 BSB102 BSB110 BSB108 CFA23 ISB41 BSB103 BSA40 BSB99 CFA16 CFA18 CFA3 ISB30 BSA33 ISB29 CFS200 ISA2 CFA24 BSB105 CFA37 ISS141 BSA95 CFA10 ISA4 BSA82 CFA25 BSB111 ISI64 BSB117 ISA20 CFA19 ISA6 ISA3 BSB100 ISB31 ISB38 ISB26 BSA210 CFA20 CFA35 ISA17 ISS148 BSB115 ISA9 CFA4 ISA7 CFA5 ISA22 CFA8 CFA33 CFA29 BSA30 BSA84 BSA44 BSB107 ISB37 ISA8 BSB109 ISA19 ISB36 ISA13 ISA1 BSB121 ISA14 BSB112 ISA21 ISA10 CFA11 ISA12 BSA15 BSB104 BSA92 BSB106 BSA94 ISA18 CFA17 ISI87 BSB114 ISA15 BSB116 ISB28 BSB97 CFA15 ISA11 ISB33 BSA47 ISB40 ISB39 CFA7 CFA13 ISS146 ISB25 BSA45 BSB118 CFA1 CFS191 ISB35 CFB65 CFA31 BSB113 ISB32 ISA16 CFS210 BSA48 BSA36 ISI97 CFA30 CFA2 CFB80 CFA38 CFA32 ISA5 BSA49 CFB64 __typename } query Query($ticker: String!, $period: String!) { CompanyFinancialRatio(ticker: $ticker, period: $period) { ratio { ...Ratios __typename } period __typename } }"

func GetReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol, err := request.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	period := request.GetString("period", "Q")
	// effectivePeriod := FINANCIAL_REPORT_PERIOD_MAP[period]
	// if effectivePeriod == "" {
	// 	return mcp.NewToolResultError("invalid period"), nil
	// }

	// Create payload struct
	payload := GraphQLPayload{
		Query: query,
		Variables: map[string]interface{}{
			"ticker": symbol,
			"period": period,
		},
	}

	// Marshal to JSON bytes
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	http_req, err := http.NewRequestWithContext(ctx, "POST", GRAPHQL_URL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Write headers from VCI_HEADERS
	for k, v := range VCI_HEADERS {
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
	reader, err := GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
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

	fmt.Println(result)

	// Extract data
	data := result["data"].(map[string]interface{})
	report := data["CompanyFinancialRatio"].(map[string]interface{})
	fmt.Println(report)
	return mcp.NewToolResultStructuredOnly(report), nil
}
