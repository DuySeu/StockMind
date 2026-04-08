# Requirements: Milestone v2.0 Agent & MCP Refactor

## 1. SDK Core & Infrastructure
- [ ] **SDK-01**: Migrate `internal/agent` to use official `github.com/anthropics/anthropic-sdk-go`.
- [ ] **SDK-02**: Ensure `factory.go` remains untouched while providing `*anthropic.Client` to providers.
- [ ] **SDK-03**: Remove manual HTTP/JSON calls in favor of SDK methods (`Messages.NewStreaming`).
- [ ] **SDK-04**: Standardize `StopReason` mapping between Anthropic SDK and internal `database.StopReason`.

## 2. Streaming & Content Handling
- [ ] **STRM-01**: Implement robust streaming for `text` content blocks.
- [ ] **STRM-02**: Implement streaming for `thinking` blocks (reasoning) with proper `ChatEvent` emitted to the UI.
- [ ] **STRM-03**: Implement accumulation of partial thinking/text to reconstruct full messages for database storage.
- [ ] **STRM-04**: Fix nil-pointer dereferences when accessing content blocks in the stream loop.

## 3. MCP Tool-Calling Workflow
- [ ] **MCP-01**: Implement robust `tool_use` block detection in the SDK stream.
- [ ] **MCP-02**: Buffer `input_json_delta` and only unmarshal full JSON at `content_block_stop`.
- [ ] **MCP-03**: Fix tool name parsing (splitting `mcp--tool_name`) to route to correct MCP clients.
- [ ] **MCP-04**: Ensure `user_id` and `session_id` are propagated to MCP tool calls via metadata.
- [ ] **MCP-05**: Handle `ToolResult` conversion from MCP output back to SDK `ToolResultBlockParam` format.

## 4. Stability & Best Practices
- [ ] **STB-01**: Fix "incorrect number of arguments" in callback calls across `anthropic_provider.go`.
- [ ] **STB-02**: Resolve log formatting errors (missing `%v` or extra arguments).
- [ ] **STB-03**: Ensure `SetAgent` correctly handles back-references without causing race conditions or deadlocks.
- [ ] **STB-04**: Standardize error handling in `AnthropicProvider` to prevent panics on unexpected stream events.

## Out of Scope
- Migrating `OpenAIProvider` to its official SDK (this milestone focuses strictly on the Anthropic/MCP path).
- Changing the React frontend's message structure.
- Modifying `internal/database` generated code.

## Future Requirements (v2.1+)
- Hybrid Search (RAG).
- Multi-user Knowledge Isolation.
- Citation tracking.
