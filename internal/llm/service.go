package core

import (
	"context"
	"log/slog"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/models"
	"stockmind/internal/llm/tools"
)

// LLMService handles LLM interactions (agentic tool loop + summarization).
// It has no database dependency — all persistence is handled by the caller.
type LLMService struct {
	tools    *tools.Manager
	provider models.Provider
}

func NewLLMService(ctx context.Context, cfg common.LLM, toolMgr *tools.Manager) (*LLMService, error) {
	provider, err := models.NewProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &LLMService{
		tools:    toolMgr,
		provider: provider,
	}, nil
}

// StructuredCompletion sends a prompt to the LLM and unmarshals the JSON response into result.
func (s *LLMService) StructuredCompletion(ctx context.Context, prompt string, result any) error {
	return s.provider.StructuredCompletion(ctx, prompt, result)
}

// ──────── Agentic Tool Loop ────────

// runToolRound runs each tool in order, streams ToolResult events, and appends
// one assistant history row containing every tool call/output pair.
func (s *LLMService) runToolRound(ctx context.Context, outputCh chan<- database.StreamEvent, history []database.Message, pending []database.Tool) []database.Message {
	assembled := make([]database.Tool, 0, len(pending))
	for _, tc := range pending {
		result, execErr := s.tools.Execute(ctx, tc.Name, tc.Arguments)
		isError := "false"
		if execErr != nil {
			result = execErr.Error()
			isError = "true"
		}
		tr := database.Tool{ID: tc.ID, Result: result, IsError: isError}
		outputCh <- database.StreamEvent{Type: database.EventToolResult, Data: tr}
		// Preserve Name + Arguments so the next provider round can reconstruct a
		// valid assistant tool_call message (an empty function name is rejected
		// by OpenRouter/OpenAI/Anthropic, breaking the follow-up completion).
		assembled = append(assembled, database.Tool{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
			Result:    tr.Result,
			IsError:   tr.IsError,
		})
	}
	meta := []database.Metadata{{Tool: assembled}}
	return append(history, database.Message{Role: "assistant", Metadata: meta})
}

// LLMOptions holds optional parameters for LLMChat.
type LLMOptions struct {
	SystemPrompt string
	// Tools restricts this turn to the named tools. Nil (the zero value) means
	// every registered tool, which is what the chat path wants; a non-nil slice
	// means exactly those tools, and a non-nil empty slice means none at all.
	// See tools.Manager.Subset.
	Tools []string
}

// LLMChat runs the agentic tool loop: call LLM → execute tool calls → append results → repeat until done.
// It has no database interactions or session management.
func (s *LLMService) LLMChat(ctx context.Context, history []database.Message, opts ...LLMOptions) (<-chan database.StreamEvent, error) {
	outputCh := make(chan database.StreamEvent, 4)

	// Options are variadic, so callers may omit them entirely — take the first if
	// present and otherwise fall back to the zero value (all tools, no system prompt).
	var opt LLMOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	go func() {
		defer close(outputCh)
		toolDefs := s.tools.Subset(opt.Tools)
		round := 0

	nextProviderRound:
		for {
			round++
			slog.Info("llm: provider round start", "round", round, "history", len(history), "tools", len(toolDefs))
			streamCh, err := s.provider.Completion(ctx, history, toolDefs, opt.SystemPrompt)
			if err != nil {
				slog.Error("llm: completion failed", "round", round, "error", err)
				outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
				return
			}

			var pendingTools []database.Tool
			var textLen, thinkingLen int // diagnostic: bytes streamed this round

			for event := range streamCh {
				switch event.Type {
				case database.EventText:
					textLen += len(event.Content)
					outputCh <- event

				case database.EventThinking:
					// NOTE: the loop does not forward thinking deltas today; we only
					// count them here to see if a round's answer arrives as reasoning.
					thinkingLen += len(event.Content)

				case database.EventToolCall:
					tc := event.Data.(database.Tool)
					outputCh <- event
					pendingTools = append(pendingTools, tc)

				case database.EventDone:
					slog.Info("llm: provider round done", "round", round,
						"text_len", textLen, "thinking_len", thinkingLen, "tool_calls", len(pendingTools))
					if len(pendingTools) > 0 {
						history = s.runToolRound(ctx, outputCh, history, pendingTools)
						pendingTools = pendingTools[:0]
						goto nextProviderRound
					}
					outputCh <- event
					return

				case database.EventError:
					slog.Error("llm: provider round error", "round", round, "error", event.Data)
					outputCh <- event
					return
				}
			}

			// Stream closed without an explicit EventDone.
			slog.Warn("llm: stream closed without done", "round", round,
				"text_len", textLen, "thinking_len", thinkingLen, "pending_tools", len(pendingTools))
			if len(pendingTools) > 0 {
				history = s.runToolRound(ctx, outputCh, history, pendingTools)
				continue
			}
			return
		}
	}()

	return outputCh, nil
}
