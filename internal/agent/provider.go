package agent

import (
	"context"
	"stockmind/internal/database"

	"github.com/anthropics/anthropic-sdk-go"
	openai "github.com/sashabaranov/go-openai"
)

type ChatEvent struct {
	Type       EventType
	Content    string // Text or Thinking content
	ToolUse    ToolCallWrapper
	ToolResult ToolResultWrapper
	IsEnd      bool // Signal end of block
}

type EventType string

const (
	EventTypeText       EventType = "text"
	EventTypeThinking   EventType = "thinking"
	EventTypeToolUse    EventType = "tool_use"
	EventTypeToolResult EventType = "tool_result"
)

type ToolCallWrapper struct {
	OpenAI    openai.ToolCall
	Anthropic anthropic.MessageParam
}

type ToolResultWrapper struct {
	OpenAI    openai.ChatCompletionMessage
	Anthropic anthropic.ContentBlockParamUnion
}

type ChatCallBack func(event ChatEvent) error

type Attachment struct {
	Name      string
	MediaType string
	Data      []byte
}

// LLMProvider defines the interface that all AI providers must implement.
// This allows the Agent to work with any provider (OpenAI, Anthropic, etc.)
// without knowing the specific implementation details.
type LLMProvider interface {
	// Completion sends messages to the LLM and returns the response
	Completion(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error)

	// ToolUse processes a tool use request from the LLM
	ToolUse(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) ([]database.MessageUnion, error)

	// SetAgent sets the back-reference to the Agent
	SetAgent(agent *Agent)
}
