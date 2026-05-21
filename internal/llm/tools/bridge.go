package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"stockmind/internal/mcp"
)

// BridgeMCPTools dynamically queries all active servers from the mcp.Manager
// and returns their tool definitions bridged into StockMind's internal tools format.
func BridgeMCPTools(ctx context.Context, manager *mcp.Manager) ([]*Tool, error) {
	var bridgedTools []*Tool

	for _, serverName := range manager.ActiveServers() {
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
			description := mt.Description

			// InputSchema is of type any, which in ListTools is unmarshaled as map[string]any.
			// Let's perform a type check/conversion to ensure it's map[string]any.
			schemaMap, ok := mt.InputSchema.(map[string]any)
			if !ok {
				// Fallback to json marshal/unmarshal if it's not directly a map
				schemaBytes, err := json.Marshal(mt.InputSchema)
				if err == nil {
					_ = json.Unmarshal(schemaBytes, &schemaMap)
				}
			}

			// Capture loop variables correctly for the closure
			targetServer := serverName
			targetTool := mcpToolName
			targetClient := client

			executeFn := func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error) {
				var args map[string]any
				if len(rawArgs) > 0 && string(rawArgs) != "null" {
					if err := json.Unmarshal(rawArgs, &args); err != nil {
						return nil, fmt.Errorf("failed to unmarshal bridge arguments: %w", err)
					}
				}

				resStr, err := targetClient.CallTool(ctx, targetTool, args)
				if err != nil {
					return nil, fmt.Errorf("mcp bridge error on %s.%s: %w", targetServer, targetTool, err)
				}

				// Check if the result is already valid JSON
				var checkJSON any
				if err := json.Unmarshal([]byte(resStr), &checkJSON); err == nil {
					return json.RawMessage(resStr), nil
				}

				// If not valid JSON, marshal it as a JSON string so the LLM loop parses it correctly
				fallbackJSON, err := json.Marshal(resStr)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal non-JSON tool result: %w", err)
				}
				return fallbackJSON, nil
			}

			bridgedTools = append(bridgedTools, &Tool{
				Name:        localToolName,
				Description: description,
				Schema:      schemaMap,
				Execute:     executeFn,
			})
			log.Printf("Dynamic MCP Tool bridged: %s (routes to %s.%s)", localToolName, serverName, mcpToolName)
		}
	}

	return bridgedTools, nil
}
