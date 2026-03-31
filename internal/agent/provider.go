package agent

import (
	"context"
	"stockmind/internal/database"
)

// LLMProvider defines the interface that all AI providers must implement.
// This allows the Agent to work with any provider (OpenAI, Anthropic, etc.)
// without knowing the specific implementation details.
type LLMProvider interface {
	// Completion sends messages to the LLM and returns the response
	Completion(ctx context.Context, messages []*database.MessageUnion, callback ChatCallBack) (database.MessageUnion, database.StopReason, error)

	// ToolUse processes a tool use request from the LLM
	ToolUse(ctx context.Context, message *database.MessageUnion, callback ChatCallBack) ([]database.MessageUnion, error)
}
