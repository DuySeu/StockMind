# Research Summary: Agent & MCP Refactor (v2.0)

## Overview
This research focused on migrating the `agent` package to the official `anthropic-go-sdk` while fixing existing bugs and maintaining the current workflow structure (with an immutable `factory.go`).

## Key Findings

### 1. SDK Migration
- **Recommendation**: Transition from manual API calls to the official `anthropic-sdk-go` client.
- **Streaming**: Utilize `Messages.NewStreaming` for robust handling of text, thinking, and tool-use deltas.
- **Standardization**: Use the SDK's built-in types (`MessageParam`, `ContentBlockParamUnion`) for all message construction.

### 2. MCP Orchestration
- **Tool Discovery**: Fetch tools from `mcp-go` clients and rename with `--` delimiter to prevent collisions.
- **Execution Loop**: Keep the `SessionManager` as the primary loop orchestrator (`StopReason` driven), but harden the `Provider`'s role in detecting and parsing tool requests.

### 3. Architecture & Constraints
- **Factory.go**: Remains the source of truth for SDK client creation.
- **Provider Interface**: The `LLMProvider` interface is solid but needs consistent implementation in `AnthropicProvider` to avoid nil pointers.
- **Callback Pattern**: Standardize `ChatEvent` to allow the frontend to distinguish between "thinking" blocks and final "text" content.

### 4. Implementation Strategy
- **Phase 1**: Refactor `AnthropicProvider.Completion` to use the SDK stream correctly and handle all delta types.
- **Phase 2**: Fix `ToolUse` logic, specifically around tool name parsing and result conversion.
- **Phase 3**: Audit `Agent` and `SessionManager` for metadata propagation and error handling.

## Watch Out For
- **JSON errors**: Ensure full buffering of tool arguments before unmarshaling.
- **Nil Access**: Guard all SDK block unions before accessing specific fields.
- **Concurrency**: Maintain context propagation to allow cancellation of long-running tool calls.

---
**Status**: Research Complete. Ready to define requirements.
