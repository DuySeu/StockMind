# Research: Architecture — Integration & decoupling

## Component Hierarchy

- **`SessionManager`**: Orchestrates the multi-turn conversation and tool-calling loop. It interacts with the database for history persistence.
- **`Agent`**: The logical worker that combines a `Provider`, a set of `McpClients`, and `AgentConfig`. It handles MCP tool discovery and renaming (to avoid collisions).
- **`LLMProvider`**: Interface for LLM interactions. Decouples the `Agent` from specific SDK details.
- **`AnthropicProvider`**: Concrete implementation using the official SDK.

## Data Flow for MCP

1. `Agent.New` fetches tool definitions from `mcpClients` and stores them in `Agent.tools` with a prefix (e.g., `market--get_stock`).
2. `AnthropicProvider.Completion` sends these tool definitions to Claude.
3. If Claude responds with `tool_use: market--get_stock`, the `Provider` emits a `StopReasonToolCall`.
4. `SessionManager` calls `Agent.ToolUse`.
5. `Agent` (via `Provider`) splits `market--get_stock` back into client `market` and tool `get_stock`.
6. Results are fed back into the next `Completion` call.

## Constraint Handling: `factory.go`
- `factory.go` is responsible for client creation. It returns an `*anthropic.Client`.
- The `Provider` will continue to receive this client from the `SessionManager` during initialization.
- No changes needed to `factory.go` as long as it returns a valid, configured client.
