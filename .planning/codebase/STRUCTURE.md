# STRUCTURE

## Subdirectories breakdown

- `/cmd`
  - `/cmd/main.go` - The main entry point of the Go backend. CLI commands attached via `urfave/cli` to start server/worker.

- `/internal`
  - Core of the backend logic.
  - `/internal/agent` - Intelligent Assistant state machine and LLM connectivity components.
  - `/internal/common` - Shared utilities, loggers, or interfaces.
  - `/internal/mcp` - Model Context Protocol servers/handlers implementation for AI tools.
  - `/internal/server` - API routes (`chi` implementations).
  - `/internal/service` - Main domain services connecting different components.
  - `/internal/database` - Storage interface, SQLC auto-generated code and connection pools.

- `/schema`
  - `/schema/migrations` - SQL scripts managed by Goose to alter database schemas.

- `/frontend`
  - `/frontend/src` - Primary React codebase.
  - `/frontend/src/api` - Client side API abstraction.
  - `/frontend/src/components` - Reusable UI widgets (RadixUI based).
  - `/frontend/src/hooks` - Stateful React logic abstractions.
  - `/frontend/src/pages` - Routable React Views and screens.
  - `/frontend/src/router` - Client side routing index.

- `/.agent`
  - GSD tools metadata, agent skills, and workflow templates.

- `docker-compose.yml` & `Dockerfile`
  - Orchestration files detailing container setups and environment bindings.
