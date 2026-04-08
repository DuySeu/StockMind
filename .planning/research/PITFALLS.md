# Research: Pitfalls — Common Mistakes & Prevention

## 1. JSON Partial Unmarshaling
- **Problem**: Attempting to unmarshal a partial JSON string before the stream has finished.
- **Prevention**: Buffer the `partial_json` from SDK deltas and only unmarshal once in the `content_block_stop` event (where `type == tool_use`).

## 2. Nil Pointer Dereferences
- **Problem**: Accessing `contentItem.OfThinking` or `OfText` before the block has been initialized in `content_block_start`.
- **Prevention**: Use guard clauses and ensure blocks are correctly initialized. Explicitly check for the existence of the expected block type before mutation.

## 3. Circular Dependencies
- **Problem**: `Agent` needs `Provider`, and `Provider` might need `Agent` (e.g., to access `mcpClients`).
- **Prevention**: Use the `SetAgent` method in the `LLMProvider` interface to establish the back-reference after both are instantiated, or pass necessary dependencies (like `mcpClients` map) into the Provider methods.

## 4. Callback Timing & Deadlocks
- **Problem**: Calling the `ChatCallback` synchronously inside the streaming loop might block the stream if the callback is slow or if it triggers another operation on the same context.
- **Prevention**: Ensure callbacks are lightweight. Consider using non-blocking channels if the callback logic grows complex.

## 5. Metadata Mapping
- **Problem**: Loss of `user_id` or `session_id` when calling MCP tools.
- **Prevention**: Explicitly extract metadata from the session and pass it into the `mcp.CallToolRequest` parameters / meta.
