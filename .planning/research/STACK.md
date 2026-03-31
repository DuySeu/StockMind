# STACK

## Standard Backend Stack (Go 1.25.1 focus)

**Core Frameworks**
- **Routing**: `go-chi/chi/v5` is ideal. Standard library compatible, lightweight, extremely fast.
- **Database**: PostgreSQL with `jackc/pgx/v5`. High performance, native types, best-in-class connection pooling.
- **SQL Generation**: `sqlc` (already present). Preferred over ORMs like GORM for performance and raw SQL predictability.
- **Websockets**: `coder/websocket` is currently the modern recommended library (replaces gorilla/websocket).

**Security Additions (Recommended)**
- **Rate Limiting**: `go-chi/httprate`. Native chi integration, configurable backoff.
- **CORS/Headers**: `go-chi/cors` and manual secure headers setup (`X-Frame-Options`, `Strict-Transport-Security`).
- **Secret Management**: Move away from raw `.env` defaults or purely hardcoded structs. Abstract into a Configuration singleton loaded via `kelseyhightower/envconfig` or `joho/godotenv` with strict struct validation.

**Performance & Telemetry**
- **Logging**: `log/slog` (Standard in Go 1.21+). Structured JSON logging out of the box. Do not use Zap or Logrus unless absolute millisecond performance is needed.
- **Tracing**: OpenTelemetry (`go.opentelemetry.io/otel`). Useful to trace LLM request latency and database bottlenecks.

**What NOT to use**
- **Heavy ORMs** (e.g., GORM): They obscure SQL performance, hide N+1 query issues, and are harder to secure.
- **Heavy DI Frameworks** (e.g., Google Wire, Uber FX): For this scale, Manual Dependency Injection (passing dependencies to structs) yields cleaner, more understandable code.
