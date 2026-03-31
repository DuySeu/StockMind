# PITFALLS

## Common Go Backend Refactoring Mistakes

**1. Context Leaks in Goroutines (Memory Leak)**
- **Warning Sign**: Memory usage scales up relative to websocket connections or LLM stream requests and never drops after disconnects.
- **Prevention Strategy**: Always pass HTTP `r.Context()` deeply into LLM processes. If initiating background jobs, use `context.WithTimeout` or `context.WithCancel` and explicitly defer cancellation.
- **Phase Mapping**: Addressed during the AI Orchestration decouple phase.

**2. Shadowing Database Contexts**
- **Warning Sign**: Database locks or transaction pools exhausting quickly under load.
- **Prevention Strategy**: Ensure `pgx` runs with proper Max/Min connection pool sizing (`ParseConfig`). Ensure transactions (`tx`) are strictly managed with `defer tx.Rollback()` implementations.
- **Phase Mapping**: Addressed during Database/Service boundary phase.

**3. Tightly Coupled Testing (Inability to Mock)**
- **Warning Sign**: Unit tests require a live Postgres instance and an active OpenAI key.
- **Prevention Strategy**: Depend exclusively on Interfaces for all external SDKs and Database layers. Generate mocks via `go:generate` or manual wrapper structs. 
- **Phase Mapping**: Built iteratively across all layers.

**4. Panics in Production**
- **Warning Sign**: App crashes totally because of a single unhandled nil pointer.
- **Prevention Strategy**: Standardize `Recoverer` middleware in Chi. Eliminate bare `goroutine` launches without internal panic catching (`defer func() { recover() }`).
