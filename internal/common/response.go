package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"stockmind/internal/database"
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
func SSEEvent(eventType database.StreamEventType, data any) map[string]any {
	return map[string]any{"type": eventType, "data": data}
}

// WriteSSE JSON-encodes v and writes it as a single `data:` SSE frame, then
// flushes. The returned error is the write error: once the client is gone every
// subsequent write fails, and a relay loop that ignores that keeps generating
// into a dead socket.
func WriteSSE(w http.ResponseWriter, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	FlushSSE(w)
	return nil
}

// WriteSSEComment emits an SSE comment frame (`: text`). A comment carries no
// `data:` line, so clients ignore it; it exists purely so an idle connection
// keeps producing bytes and isn't reaped by a proxy, load balancer, or browser
// idle timeout while the model reasons or a slow tool runs.
func WriteSSEComment(w http.ResponseWriter, text string) error {
	if _, err := fmt.Fprintf(w, ": %s\n\n", text); err != nil {
		return err
	}
	FlushSSE(w)
	return nil
}
