# FEATURES (Refactoring & Security Dimension)

## Table Stakes (Must Have for Production)
- **Graceful Shutdown**: Essential for keeping websocket connections safe and database transactions complete during rollouts.
- **Secure Context Cancellation**: Preventing goroutine memory leaks specifically around long-running LLM API streaming calls and WebSocket disconnects.
- **API Security & Limits**: Basic payload size limits, connection timeouts (`ReadHeaderTimeout`), and endpoint rate-limiting to prevent DDoS or runaway LLM costs.
- **Structured Error Handling**: A centralized error handler that does not leak internal db errors to the frontend React app.

## Differentiators
- **LLM Circuit Breaking**: Implement resilience (e.g., `sony/gobreaker`) over OpenAI/Anthropic APIs so the app fails gracefully if the LLM provider drops.
- **Caching Layer**: Basic in-memory caching or Redis abstraction to cache expensive financial analyses (Piotroski computations) reducing redundant DB or API reads.

## Anti-Features (Do Not Build)
- **Frontend Refactoring**: The UI should remain completely unbothered, save for updated API resilience.
- **Database Engine Swap**: Stick with PostgreSQL, don't migrate simply for "scale" without proving pgx has failed.
