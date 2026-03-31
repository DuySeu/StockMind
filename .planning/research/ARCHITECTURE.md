# ARCHITECTURE 

## Layered Component Architecture

A production-ready decoupled Go application typically follows a 3 or 4-layer architecture mapping:

**1. Transport/Routing Layer (`internal/server`)**
- Only handles HTTP decoding, validation, and JSON encoding.
- Interacts with Services, holds NO business logic.
- Maintains `slog` middleware, CORS, and Rate Limiting.

**2. Application/Service Layer (`internal/service` & `internal/agent`)**
- Represents core business workflows.
- Contains the LLM Orchestrator logic.
- Communicates only via predefined Interfaces, never directly to the database or external APIs.

**3. Integration & Data Access Layer (`internal/database` & `internal/mcp`)**
- Pure Database repositories (via `sqlc` models).
- Third-party API Clients (Anthropic, OpenAI, MCP protocol).

## Data and Execution Flow
1. Request arrives at `Chi` router. Middleware attaches request ID and `slog` logger to Context.
2. HTTP Handler parses input, calls `Service` methods passing the `context.Context`.
3. `Service` manages any Goroutines, explicitly passing cancellable contexts.
4. `Service` queries the `Repository` (DB) or `Client` (LLM via MCP) through Interfaces.
5. Result flows back cleanly; errors are translated to standardized API errors.
