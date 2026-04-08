# Research: Features — Streaming & MCP Orchestration

## Streaming Architecture

### Content Types:
- **Text**: Standard textual response.
- **Thinking**: Extended internal reasoning (for `claude-3-7-sonnet`).
- **Tool Use**: Partial JSON for tool arguments.

### Event Handling:
- **`content_block_start`**: Initialize the specific block type (Text, Thinking, or ToolUse).
- **`content_block_delta`**: Append content. For `ToolUse`, append to a `partial_json` string.
- **`content_block_stop`**: Finalize the block. Crucial for `ToolUse` to unmarshal the final JSON.

## MCP Orchestration Loop

1. **Detection**: Identifying `tool_use` blocks in the assistant's stream.
2. **Suspension**: The `Completion` call ends when a tool use is detected (or turn ends).
3. **Execution**: `SessionManager` detects `StopReasonToolCall` and invokes `ToolUse`.
4. **Resolution**: `ToolUse` fetches results from sub-MCP-clients and produces a `user` message with `tool_result` content.
5. **Resumption**: `SessionManager` sends the history (including the `tool_result`) back to the agent.
