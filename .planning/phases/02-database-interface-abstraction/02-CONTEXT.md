# Phase 2: Database Interface Abstraction - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Introduce strict boundaries between `sqlc` models and the `service` layers by wrapping DB methods inside explicit repository interfaces. This prevents hard-coupling to auto-generated signatures.
</domain>

<decisions>
## Implementation Decisions

### 1. Database Wrapper Placement
- **D-01:** **Ad-hoc Interface Definition** (User Selected) — We will NOT create a centralized repository package. Interfaces matching database methods will be defined exactly where they are consumed (e.g., `agent.DB` or `server.UserStore`). The `*database.Queries` instance will be passed directly but cast to these interfaces automatically.

### 2. Transaction Handling Strategy
- **D-02:** **Explicit Wrapper Methods** (Auto-Selected) — When a cross-cutting transaction is required, the repository layer should expose functional closures like `WithTransaction(ctx, func(repo Repository) error)` rather than trying to pass `pgx.Tx` through the contexts cleanly, preventing transaction leakages.

### 3. Data Mapping Protocol
- **D-03:** **Return Output Structs Direct** (Auto-Selected) — At this phase size, we will NOT map generated `database.User` structures to a purely blank domain structural layer. The `sqlc` models are clean enough to act as the primary struct carriers returned by the repository interfaces. (This simplifies the refactor while still securing the method signatures).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture
- `.planning/codebase/ARCHITECTURE.md` — Reaffirms the need for the Data Access Layer to have distinct interface wrappers around generated objects.
- `.planning/REQUIREMENTS.md` — [ARCH-02] Requirement definition mapping to Phase 2.
- `internal/database/db.go` — Current `Queries` structs that need to be wrapped.
</canonical_refs>

<deferred>
## Deferred Ideas

*(None)*
</deferred>
