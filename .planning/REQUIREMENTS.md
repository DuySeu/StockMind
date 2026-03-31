# Requirements

## v1 Requirements

### Architecture (ARCH)
- [ ] **ARCH-01**: Implement strict Dependency Injection to standardize how backend services receive dependencies.
- [ ] **ARCH-02**: Standardize the Database Interface to fully isolate generated `sqlc` models from application business logic.

### Reliability (RELI)
- [ ] **RELI-01**: Enforce secure context cancellations across all HTTP, WebSocket, and LLM orchestration layers to prevent Goroutine memory leaks and database connection exhaustion.

## v2 Requirements (Deferred)
- **Graceful Shutdown**: Safely terminate long-running processes on deploy.
- **API Security / Limits**: Request rate-limiting, strict timeouts.
- **Structured Error Handling**: Unified Go error wrapper for HTTP masking.
- **Full Backend Mocks/Tests**: Unit tests using the newly standardized interfaces.
- **LLM/MCP Subpackage Splits**: Extracting logic from general routers entirely.

## Out of Scope
- **Frontend Refactoring** — Scope isolated entirely to Go backend.
- **Caching Layer / Redis** — Too complex for standard v1 stabilization.
- **LLM Circuit Breaking** — Resiliency external to context timeouts is deferred.

## Traceability
- **ARCH-01**: Maps to Phase 1
- **ARCH-02**: Maps to Phase 2
- **RELI-01**: Maps to Phase 3
