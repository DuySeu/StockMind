# TESTING

## Backend
- The primary test framework is Go's built-in `testing` library.
- Used for internal application unit tests (`go test ./...`).
- Integration tests targeting PostgreSQL run exclusively using `make itest`. This specifically targets database connectivity/health in isolated scenarios mapped to `internal/database`.

## Frontend
- The `eslint` rule (`npm run lint`) enforces code style checks to identify syntactical/anti-pattern anomalies before transpilation.
- No active Unit Test runner (e.g., Jest/Vitest) is currently set up over the component trees.

## Testing Setup
- Environment configuration is simulated locally via standard `.env` overriding mechanisms prior to container booting, but `Makefile`/`docker-compose` ensures testing against a live/sandboxed container instance rather than in-memory SQLite mapping.
