package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	ctx := context.Background()

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	fmt.Printf("Current working directory: %s\n", wd)

	// Check if the MCP main.go file exists
	mcpPath := filepath.Join(wd, "cmd", "mcp", "main.go")
	if _, err := os.Stat(mcpPath); err != nil {
		log.Fatalf("MCP main.go not found at %s: %v", mcpPath, err)
	}
	fmt.Printf("MCP main.go found at: %s\n", mcpPath)

	// Create the transport with absolute path
	fmt.Println("Creating MCP transport...")
	tr := transport.NewStdio("go", []string{"run", mcpPath})

	// Create client with the transport
	fmt.Println("Creating MCP client...")
	cli := client.NewClient(tr)

	// Start the client
	fmt.Println("Starting MCP client...")
	if err := cli.Start(ctx); err != nil {
		log.Fatalf("Failed to start MCP client: %v", err)
	}
	defer cli.Close()

	// Initialize the client
	fmt.Println("Initializing MCP client...")
	_, err = cli.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		log.Fatalf("Failed to initialize MCP client: %v", err)
	}

	// List available tools
	fmt.Println("Listing MCP tools...")
	toolsResp, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("MCP Server is working! Found %d tools:\n", len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Test calling a tool
	if len(toolsResp.Tools) > 0 {
		fmt.Println("\nTesting hello_world tool...")
		result, err := cli.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "hello_world",
				Arguments: map[string]interface{}{"name": "Test User"},
			},
		})
		if err != nil {
			log.Fatalf("Failed to call tool: %v", err)
		}
		fmt.Printf("Tool result: %v\n", result.Content)
	}
}
