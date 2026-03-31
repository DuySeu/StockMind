# Phase 1: Dependency Injection Assembly - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish manual Dependency Injection across all Go packages to standardize initialization and remove global states.
</domain>

<decisions>
## Implementation Decisions

### Tool Configuration
- **D-01:** LLM Configurations (`DefaultOpenAIConfig` and `DefaultAnthropicConfig`) will be removed from global blocks. They will be explicitly initialized in `main.go` and injected via `agent.NewService(..., config)` natively during startup.

### the agent's Discretion
- **D-02:** Injection Pattern: The planner should specify exactly how dependencies are passed (e.g. passing an `App` container or individual explicit parameters to constructors like `NewServer`).
- **D-03:** Configuration management structure (whether to use a single `Config` struct abstraction or dynamically read env during bootstrap) is left to the planner.
- **D-04:** State Managers: Refactoring other Singletons (like `GlobalStreamManager`) is delegated to the planner, following the paradigm established in D-01.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture
- `.planning/codebase/ARCHITECTURE.md` — Explains the layer abstraction goals (Transport, Service, Data).
- `.planning/codebase/CONVENTIONS.md` — Enforces "Effective Go rules: Small readable functions, interface adherence, explicit package visibility."
- `cmd/server/main.go` — Current initialization entry point that must be updated to inject configs.
- `internal/agent/configs.go` — Current location of globals that serve as the main refactoring targets for this phase.
</canonical_refs>

<deferred>
## Deferred Ideas

*(None)*
</deferred>
