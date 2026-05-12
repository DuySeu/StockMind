package core

type StreamEventType string

const (
	EventThinking   StreamEventType = "thinking"
	EventText       StreamEventType = "text"
	EventToolCall   StreamEventType = "tool_call"
	EventToolResult StreamEventType = "tool_result"
	EventError      StreamEventType = "error"
	EventDone       StreamEventType = "done"
)

type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Content string          `json:"content,omitempty"` // For text delta
	Data    any             `json:"data,omitempty"`    // For Error or ToolCall/Result details
}
