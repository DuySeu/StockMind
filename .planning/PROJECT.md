# StockMind - Backend Audit & Refactor

## What This Is

StockMind is an AI-powered financial assistant tailored for the Vietnamese stock market. This project phase focuses specifically on auditing and refactoring the existing Go backend to eliminate messy package structures, decouple logic, resolve concurrency/memory issues, and harden security for production readiness.

## Core Value

A reliable, scalable, and secure Go backend foundation that handles seamless financial data retrieval and AI orchestrations without performance bottlenecks or memory leaks.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->
- ✓ Go backend running with Chi router and PostgreSQL integration — existing
- ✓ LLM integrations (Anthropic/OpenAI) via internal AI orchestration — existing
- ✓ MCP (Model Context Protocol) external tool integration — existing
- ✓ Core financial evaluation frameworks — existing

### Active

<!-- Current scope. Building toward these. -->
- [ ] Refactor messy and tightly coupled Go logic into clean, distinct boundaries.
- [ ] Identify and resolve any Goroutine concurrency issues or memory leaks.
- [ ] Enhance backend security (API security, rate limiting, secure headers, etc.) for a production-ready state.
- [ ] Review and optimize database schema/queries as needed during the Go logic refactor.

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->
- [React Frontend Optimization] — This phase explicitly targets the Go backend first to stabilize the foundation before touching UI performance.
- [New Features] — Adding entirely new Assistant capabilities is deferred until the backend is fully stabilized.

## Context

The application already works conceptually but has accrued technical debt in its internal logic layout and state management. The backend is built natively with Go 1.25.1. Relevant existing constraints can be found internally mapped inside the `.planning/codebase/` directory (STACK.md, ARCHITECTURE.md).

## Constraints

- **Compatibility**: Architectural refactoring must not introduce breaking changes to the REST or WebSocket API contracts consumed by the current React frontend.
- **Tech Stack**: Keep dependencies aligned to Go 1.25.1 and the existing primary tools (`pgx`, `sqlc`, `goose`).

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->
| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Priority on Go Backend | Fixing backend memory and structural coupling provides the highest stability ROI before attempting UI or full database migrations. | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-31 after initialization*
