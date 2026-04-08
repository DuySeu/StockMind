# Roadmap: Milestone v2.0 Agent & MCP Refactor

## Summary
Building a robust agent foundation using the official Anthropic Go SDK and fixing core MCP orchestration bugs.

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 7 | **SDK Migration & Streaming** | Pivot to official SDK for all streaming interactions. | SDK-01..04, STRM-01..04 | Clean text/thinking streams; no nil-pointer panics. |
| 8 | **MCP Loop Execution** | Fix tool detection, unmarshaling, and metadata. | MCP-01..05 | Reliable tool calls; correct tool_result conversion. |
| 9 | **Hardening & Audit** | Resolve inconsistencies and log/callback errors. | STB-01..04 | 0 log errors; consistent callback signatures. |

## Phase Details

### Phase 7: SDK Migration & Streaming
- [ ] **SDK-01**: Refactor `AnthropicProvider.Completion` to use `client.Messages.NewStreaming`.
- [ ] **STRM-02**: Correctly handle thinking blocks for reasoning models.
- [ ] **STRM-03**: Accumulate content for database history before the Turn ends.

### Phase 8: MCP Loop Execution
- [ ] **MCP-01**: Detect `tool_use` blocks early to stop streaming gracefully.
- [ ] **MCP-02**: Finalize JSON input at the correct event signal.
- [ ] **MCP-04**: Extract `user_id` and `session_id` from the current session for tool context.

### Phase 9: Hardening & Audit
- [ ] **STB-01**: Audit all `ChatCallBack` calls in `anthropic_provider.go`.
- [ ] **STB-02**: Fix log parameter mismatches.
- [ ] **STB-03**: Verify `SetAgent` initialization sequence.
