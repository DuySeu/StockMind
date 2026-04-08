# Phase 7: SDK Migration & Streaming - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Pivot the core agent engine to use the official Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`) for all streaming interactions. This includes text, thinking, and tool_use blocks.

</domain>

<decisions>
## Implementation Decisions

### Thinking (Reasoning) Blocks
- **D-01:** Stream reasoning results immediately as they happen. The UI should not wait for the turn to complete before displaying thinking process.
- **D-02:** Use `EventTypeThinking` in `ChatEvent` to signal reasoning content to the frontend.

### Content Transitions
- **D-03:** Ensure smooth transitions between content blocks (e.g., Thinking -> Text -> ToolUse).
- **D-04:** Use visual indicators for each block type to keep the user informed of the agent's current "state".

### Error Handling & Resilience
- **D-05:** If a stream connection is lost or an SDK error occurs, inform the user clearly and provide an option to "Retry Turn".
- **D-06:** Maintain partial content accumulation progress in the database even if the full turn fails, to prevent data loss.

### Codebase Consistency
- **D-07:** Preserve `factory.go` implementation. All client access must go through the existing `Provider` initialization logic.
- **D-08:** Standardize callback signature to `func(event ChatEvent) error` as defined in `provider.go`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### SDK & MCP Docs
- `internal/agent/anthropic_provider.go` — Current problematic implementation.
- `internal/agent/provider.go` — Interface and Event definitions.
- `internal/agent/session.go` — Orchestration logic.
- [Anthropic SDK Go](https://github.com/anthropics/anthropic-sdk-go) — Reference for `NewStreaming` and event types.

</canonical_refs>

<deferred>
## Deferred Ideas
- Multi-client fallback (e.g., OpenAI if Anthropic is down).
- Fine-grained UI controls for "Thinking" visibility (e.g., collapsible blocks).
</deferred>
