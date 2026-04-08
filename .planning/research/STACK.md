# Research: Stack — Anthropic Go SDK & MCP

## Official SDK: `github.com/anthropics/anthropic-sdk-go`

### Key Components:
- **`anthropic.Client`**: Core client for interacting with the Messages API.
- **`Messages.NewStreaming`**: Returns a `*MessagesStream` which is the standard for real-time output.
- **`BetaToolRunner`**: While powerful, integrating it might require a major change to `SessionManager`'s loop. Given the "minimum change" constraint, we should focus on hardening the manual stream loop first.

### Best Practices:
1. **Accumulation**: Use `message.Accumulate(event)` if we need to reconstruct the final assistant message during streaming.
2. **Type Safety**: Avoid `map[string]any` where the SDK provides typed parameters (e.g., `ToolUseBlockParam`).
3. **Beta Headers**: Ensure current beta headers for features like MCP or Thinking are handled by the SDK (it usually handles them via options).

## MCP Integration: `github.com/mark3labs/mcp-go`

### Key Components:
- **`mcp.CallToolRequest`**: Standard structure for invoking tools.
- **Dynamic Discovery**: Tools are fetched from MCP servers and prefixed (e.g., `server--tool`).

### Integration Points:
- The `Agent` should hold the `mcpClients`.
- The `Provider` should split the prefixed tool names and route to the correct client.
