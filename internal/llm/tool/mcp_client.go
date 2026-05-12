package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tidwall/gjson"
)

// MCPClient communicates with an MCP server over stdio using JSON-RPC 2.0.
type MCPClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner

	pendingResponses sync.Map
	nextID           atomic.Uint64
}

// NewMCPClient starts an MCP server subprocess and performs the initialization handshake.
func NewMCPClient(command string, args ...string) (*MCPClient, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client := &MCPClient{
		cmd:     cmd,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
	}

	go client.listen()

	// Handshake
	_, err = client.call(context.Background(), "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "llm-core-client", "version": "1.0.0"},
	})
	if err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	return client, nil
}

// GetTools fetches the list of tools exposed by the MCP server.
func (c *MCPClient) GetTools(ctx context.Context) ([]mcp.Tool, error) {
	resp, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var defs []mcp.Tool
	tools := gjson.Get(resp, "tools").Array()
	for _, t := range tools {
		var inputSchema mcp.ToolInputSchema
		schemaStr := t.Get("inputSchema").Raw
		json.Unmarshal([]byte(schemaStr), &inputSchema)
		
		defs = append(defs, mcp.Tool{
			Name:        t.Get("name").String(),
			Description: t.Get("description").String(),
			InputSchema: inputSchema,
		})
	}
	return defs, nil
}

// CallTool invokes a tool on the MCP server and returns the text output.
func (c *MCPClient) CallTool(ctx context.Context, name string, args string) (string, error) {
	var argsMap map[string]any
	json.Unmarshal([]byte(args), &argsMap)

	resp, err := c.call(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": argsMap,
	})
	if err != nil {
		return "", err
	}

	text := gjson.Get(resp, "content.0.text").String()
	return text, nil
}

func (c *MCPClient) call(ctx context.Context, method string, params any) (string, error) {
	id := c.nextID.Add(1)
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	if params == nil {
		req["params"] = map[string]any{}
	}

	data, _ := json.Marshal(req)

	resCh := make(chan string, 1)
	c.pendingResponses.Store(id, resCh)
	defer c.pendingResponses.Delete(id)

	fmt.Fprintln(c.stdin, string(data))

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resCh:
		if gjson.Get(res, "error").Exists() {
			return "", fmt.Errorf("mcp error: %s", gjson.Get(res, "error.message").String())
		}
		return gjson.Get(res, "result").Raw, nil
	}
}

func (c *MCPClient) listen() {
	for c.scanner.Scan() {
		line := c.scanner.Text()
		id := gjson.Get(line, "id").Uint()
		if ch, ok := c.pendingResponses.Load(id); ok {
			ch.(chan string) <- line
		}
	}
}

// Close shuts down the MCP server subprocess.
func (c *MCPClient) Close() {
	c.stdin.Close()
	c.cmd.Process.Kill()
}
