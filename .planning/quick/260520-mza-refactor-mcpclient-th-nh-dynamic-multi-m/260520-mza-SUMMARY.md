# Summary: Refactor mcpclient to Dynamic Multi-MCP Bridge

We have successfully implemented the Dynamic Multi-MCP Bridge in the StockMind Go backend. This allows StockMind to dynamically query any external MCP server (like AWS documentation, GitHub, Postgres, etc.) and auto-bridge their tools into StockMind's LLM Tool Loop without any Go code modifications.

## Changes Implemented

1. **`internal/mcpclient/manager.go` (NEW):**
   - Created a thread-safe `Manager` that holds multiple MCP clients, keyed by their configured name.
   - Implements lazy loading (`GetOrStart`) to spawn the MCP servers only when needed.
   - Implements graceful shutdown via `CloseAll`.

2. **`internal/mcpclient/mcpclient.go` (MODIFIED):**
   - Added the `ListTools` method to fetch remote tool definitions from any connected MCP server.

3. **`internal/llm/tools/bridge.go` (NEW):**
   - Added `BridgeMCPTools` which fetches remote tool definitions, automatically builds internal `Tool` metadata (including namespaces like `{serverName}_{toolName}` to avoid conflicts), and sets up dynamic bridge handlers that marshal/unmarshal arguments.

4. **`internal/llm/tools/schemas.go` (MODIFIED):**
   - Removed the hardcoded `awsDocs` parameter and static tool declarations.

5. **`internal/llm/tools/aws_docs.handler.go` (DELETED):**
   - Deleted this file as AWS documentation tools are now dynamically discovered and bridged automatically.

6. **`cmd/main.go` (MODIFIED):**
   - Initialized the `mcpclient.Manager` using dynamic configurations.
   - Automatically registers bridged tools in the `LLMService` tool registry.
   - Ensures all active external MCP client sessions are closed gracefully upon server shutdown.

7. **`internal/mcpclient/manager_test.go` (NEW):**
   - Added unit tests for the new `Manager` to ensure complete coverage.

## Verification & Build Results

- Verified compile/build correctness: `go build ./cmd/...` builds successfully without errors.
- Verified test suite passes: `go test ./internal/mcpclient/...` succeeds with `PASS`.
