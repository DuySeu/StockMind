# API Standards

## REST Endpoints
- Base path: `/v1/`
- Chat: POST `/v1/chat` (SSE response with direct streaming)
- Sessions: GET `/v1/sessions`, DELETE `/v1/sessions/{id}`
- Research: POST `/v1/stock/research`, POST `/v1/stock/research/stream` (SSE)
- Documents: POST/GET/DELETE `/v1/documents/`
- Stock: GET `/v1/stock/price-board`, GET `/v1/stock/watchlist`, POST `/v1/stock/add-symbol`
- News: GET `/v1/news/`
- Agent Flows: GET/POST `/v1/agent_flows/`

## Response Format
- Use `common.WriteJSON` and `common.WriteJSONError` helpers
- SSE events: `start`, `thinking`, `text`, `tool_call`, `tool_result`, `error`, `done`

## SSE Streaming
- Chat handler sets SSE headers directly, streams from LLM channel
- No intermediate pub/sub or StreamManager
- Research stream uses same SSE pattern with progress events

## Error Handling
- Return structured JSON errors with appropriate HTTP status codes
- Log errors server-side, return user-friendly messages to client
