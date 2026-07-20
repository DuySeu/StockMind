package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	kb "stockmind/internal/knowledge"
	impl "stockmind/internal/llm/tools/implementations"
	"stockmind/internal/mcp"
	"stockmind/internal/service"
)

// RegisterTools creates all native tool definitions with their dependencies via
// closures. Execution logic lives in the implementations package; this registry
// only wires handlers, descriptions, and dependencies into Tool definitions.
func RegisterTools(retriever kb.Retriever, services *service.Services) []*Tool {
	toolList := []*Tool{
		NewTool("retrieve_knowledge",
			"Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news.",
			func(ctx context.Context, input impl.RetrieveKnowledgeInput) (any, error) {
				return impl.HandleRetrieveKnowledge(ctx, retriever, input)
			},
		),

		NewTool("get_stock_price",
			"Get OHLC stock price data for a Vietnamese stock symbol from VietCap.",
			impl.HandleGetStockPrice,
		),

		NewTool("get_report",
			"Get quarterly or yearly financial report for a stock.",
			impl.HandleGetReport,
		),

		NewTool("get_news",
			"Get stock news for a query via web search.",
			func(ctx context.Context, input impl.GetNewsInput) (any, error) {
				return impl.HandleGetNews(ctx, *services.Tavily, input)
			},
		),

		NewTool("fundamental_analysis",
			"Produce a qualitative fundamental analysis of a Vietnamese stock: company overview, shareholder structure, ecosystem, business activity, economic moat, outlook, risks, and macro context. Grounds factual fields on VietCap company data and synthesizes the analytical narrative.",
			func(ctx context.Context, input impl.FundamentalAnalysisInput) (any, error) {
				return impl.HandleFundamentalAnalysis(ctx, input)
			},
		),
	}

	return toolList
}

// BridgeMCPTools queries all configured MCP servers, lists their tools, and
// returns them bridged into StockMind's internal tool format.
// Each tool's Execute closure routes calls through Manager.CallTool (retry-aware).
func BridgeMCPTools(ctx context.Context, manager *mcp.Manager) ([]*Tool, error) {
	var bridgedTools []*Tool

	for _, serverName := range manager.ConfiguredServers() {
		client, err := manager.GetOrStart(ctx, serverName)
		if err != nil {
			log.Printf("Warning: Failed to connect to MCP server %s: %v. Skipping.", serverName, err)
			continue
		}

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools for %s: %w", serverName, err)
		}

		for _, mt := range mcpTools {
			mcpToolName := mt.Name
			localToolName := fmt.Sprintf("%s_%s", serverName, mcpToolName)

			schemaMap, ok := mt.InputSchema.(map[string]any)
			if !ok {
				schemaBytes, err := json.Marshal(mt.InputSchema)
				if err == nil {
					_ = json.Unmarshal(schemaBytes, &schemaMap)
				}
			}

			// Capture for closure — route through Manager (not a raw *Client pointer).
			targetServer := serverName
			targetTool := mcpToolName

			executeFn := func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error) {
				var args map[string]any
				if len(rawArgs) > 0 && string(rawArgs) != "null" {
					if err := json.Unmarshal(rawArgs, &args); err != nil {
						return nil, fmt.Errorf("failed to unmarshal bridge arguments: %w", err)
					}
				}

				resStr, err := manager.CallTool(ctx, targetServer, targetTool, args)
				if err != nil {
					return nil, fmt.Errorf("mcp bridge error on %s.%s: %w", targetServer, targetTool, err)
				}

				// Return as-is if already valid JSON, otherwise marshal as string.
				if json.Valid([]byte(resStr)) {
					return json.RawMessage(resStr), nil
				}
				fallbackJSON, err := json.Marshal(resStr)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal non-JSON tool result: %w", err)
				}
				return fallbackJSON, nil
			}

			bridgedTools = append(bridgedTools, &Tool{
				Name:        localToolName,
				Description: mt.Description,
				Schema:      schemaMap,
				Execute:     executeFn,
			})
			log.Printf("Dynamic MCP Tool bridged: %s (routes to %s.%s)", localToolName, serverName, mcpToolName)
		}
	}

	return bridgedTools, nil
}
