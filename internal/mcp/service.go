package mcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Start(ctx context.Context, protocol string) (func(), error) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "StockMind",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "piotroski_evaluation",
		Description: "Get Piotroski evaluation for a stock",
	}, GetPiotroskiEvaluation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "altman_z_score",
		Description: "Get Altman Z-Score for a stock",
	}, GetAltmanZScore)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_report",
		Description: "Get report for a stock",
	}, GetReport)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_news",
		Description: "Get stock news for a query",
	}, GetNews)

	switch protocol {
	case "stdio":
		return func() {}, server.Run(ctx, &mcp.StdioTransport{})
	case "http", "streamablehttp":
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return server
		}, nil)
		go func() {
			if err := http.ListenAndServe("0.0.0.0:8081", handler); err != nil {
				fmt.Printf("MCP HTTP server error: %v\n", err)
			}
		}()
		return func() {}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
