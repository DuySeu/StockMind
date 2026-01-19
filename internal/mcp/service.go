package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Start(ctx context.Context, protocol string) (func(), error) {
	// Create MCP server
	s := server.NewMCPServer(
		"StockMind 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register tools
	s.AddTool(
		mcp.NewTool("get_stock_price",
			mcp.WithDescription("Get latest stock price from VCI with symbol, time frame and look back period"),
			mcp.WithString("symbol",
				mcp.Required(),
				mcp.Description("Stock symbol, e.g., HPG"),
			),
			mcp.WithString("time_frame",
				mcp.Description("Time frame, e.g., ONE_DAY, ONE_MINUTE, ONE_HOUR. Default is ONE_DAY"),
			),
			mcp.WithNumber("count_back",
				mcp.Description("Number of data points to look back. Default is 10"),
			),
		),
		GetStockPrice,
	)

	s.AddTool(
		mcp.NewTool("piotroski_evaluation",
			mcp.WithDescription("Get Piotroski evaluation for a stock"),
			mcp.WithString("symbol",
				mcp.Required(),
				mcp.Description("Stock symbol, e.g., HPG"),
			),
		),
		GetPiotroskiEvaluation,
	)

	s.AddTool(
		mcp.NewTool("altman_z_score",
			mcp.WithDescription("Get Altman Z-Score for a stock"),
			mcp.WithString("symbol",
				mcp.Required(),
				mcp.Description("Stock symbol, e.g., HPG"),
			),
		),
		GetAltmanZScore,
	)

	s.AddTool(
		mcp.NewTool("get_report",
			mcp.WithDescription("Get report for a stock"),
			mcp.WithString("symbol",
				mcp.Required(),
				mcp.Description("Stock symbol, e.g., HPG"),
			),
			mcp.WithString("period",
				mcp.Description("Financial report period, e.g., Q(Quarter), Y(Year). Default is Q"),
			),
		),
		GetReport,
	)

	// Start server based on protocol
	switch protocol {
	case "stdio":
		return func() {}, server.ServeStdio(s)
	case "http", "streamablehttp":
		// Handle both http and streamablehttp as HTTP server
		h := server.NewStreamableHTTPServer(s)
		go func() {
			if err := h.Start("0.0.0.0:8081"); err != nil {
				fmt.Printf("MCP HTTP server error: %v\n", err)
			}
		}()
		return func() {
			// server.NewStreamableHTTPServer doesn't expose a Shutdown method easily accessible here without more context,
			// but for this implementation it runs until process termination.
		}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
