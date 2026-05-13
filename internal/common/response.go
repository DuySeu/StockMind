package common

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON writes a JSON response with the given status code and data
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeJSONError writes a JSON error response with the given status code and message
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// ──────────────────────────────────────────────────────────────────────────────
// SSE helpers
// ──────────────────────────────────────────────────────────────────────────────

// FlushSSE pushes buffered response data to the client. Uses
// [http.ResponseController] so it unwraps middleware wrappers that embed
// [http.Flusher] (e.g. chi's response writer).
func FlushSSE(w http.ResponseWriter) {
	_ = http.NewResponseController(w).Flush()
}

// SSEEvent builds the standard `{type, data}` envelope shared across streaming handlers.
func SSEEvent(eventType StreamEventType, data any) map[string]any {
	return map[string]any{"type": eventType, "data": data}
}

// WriteSSE JSON-encodes v and writes it as a single `data:` SSE frame, then flushes.
func WriteSSE(w http.ResponseWriter, v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(w, "data: %s\n\n", data)
	FlushSSE(w)
}
