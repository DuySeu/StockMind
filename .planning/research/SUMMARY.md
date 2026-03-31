# SUMMARY

## Research Synthesis: Go Backend Hardening & Refactor

**Core Objective**: Transform the existing Go backend into a highly concurrent, memory-safe, and decoupled production service without altering the frontend React consumption.

### Key Learnings

1. **Stack Simplification**: We do not need heavy frameworks to fix coupling. Standardized interfaces, manual Dependency Injection, and rigorous use of `context.Context` will resolve most issues. The addition of `go-chi/httprate` and Go 1.21 `slog` will modernize security and observability instantly.
2. **Goroutines and Streaming**: The primary source of memory leaks in LLM/Websocket apps is orphaned goroutines. The refactor MUST prioritize thread-safe cancellations originating from the HTTP layer down into the MCP orchestrators.
3. **Resilience**: The backend needs circuit-breakers or basic backoff layers around its integrations (Anthropic/OpenAI) because remote LLM latency failures can stack and crash the local Go host if connections aren't aggressively timed out.
4. **Decoupling strategy**: The transition should introduce specific interface boundaries (Router -> Service -> Repository) to completely isolate HTTP handling from core analytical evaluations. This makes the LLM interactions fully testable offline.

**Next Immediate Step**: Scope these findings into distinct actionable roadmap phases inside `REQUIREMENTS.md` and `ROADMAP.md`.
