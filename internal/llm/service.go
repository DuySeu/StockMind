package core

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"stockmind/internal/database"
	"stockmind/internal/llm/tool"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
)

type completionFunc func(context.Context, []database.Message, []mcp.Tool) (<-chan StreamEvent, error)

type LLMService struct {
	queries    *database.Queries
	tools      *tool.ToolManager
	completion completionFunc
}

func NewLLMService(ctx context.Context, providerName database.ModelProvider, model string, cfg LLMProviderConfig, q *database.Queries, t *tool.ToolManager) (*LLMService, error) {
	var completion completionFunc
	switch providerName {
	case database.ModelProviderOpenAI:
		client, err := NewOpenAIClient(cfg.OpenAI)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []mcp.Tool) (<-chan StreamEvent, error) {
			return OpenAICompletion(client, model, ctx, history, tools)
		}
	case database.ModelProviderAnthropic:
		client, err := NewAnthropicClient(ctx, cfg.Anthropic)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []mcp.Tool) (<-chan StreamEvent, error) {
			return AnthropicCompletion(client, model, ctx, history, tools)
		}
	case database.ModelProviderOpenRouter:
		client, err := NewOpenRouterClient(cfg.OpenRouter)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []mcp.Tool) (<-chan StreamEvent, error) {
			return OpenRouterCompletion(client, model, ctx, history, tools)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %q", providerName)
	}

	return &LLMService{
		queries:    q,
		tools:      t,
		completion: completion,
	}, nil
}

// runToolRound runs each tool in order, streams ToolResult events, and appends
// one assistant history row containing every tool call/output pair.
func (s *LLMService) runToolRound(ctx context.Context, outputCh chan<- StreamEvent, history []database.Message, pending []database.Tool) []database.Message {
	assembled := make([]database.Tool, 0, len(pending))
	for _, tc := range pending {
		result, execErr := s.tools.Execute(ctx, tc.Name, tc.Arguments)
		isError := "false"
		if execErr != nil {
			result = execErr.Error()
			isError = "true"
		}
		tr := database.Tool{ID: tc.ID, Output: result, IsError: isError}
		outputCh <- StreamEvent{Type: EventToolResult, Data: tr}
		assembled = append(assembled, database.Tool{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
			Output:    tr.Output,
			IsError:   tr.IsError,
		})
	}
	meta := []database.Metadata{{Tool: assembled}}
	return append(history, database.Message{Role: "assistant", Metadata: meta})
}

func (s *LLMService) Chat(ctx context.Context, sessionID uuid.UUID, userPrompt string) (<-chan StreamEvent, error) {
	// Unbuffered: mỗi event LLM chỉ rời relay khi HTTP handler đã nhận (ít gom trong RAM).
	outputCh := make(chan StreamEvent)

	go func() {
		defer close(outputCh)

		// 1. Load prior history from DB
		var history []database.Message
		if s.queries != nil {
			var err error
			history, err = s.queries.GetSessionHistoryBySessionID(ctx, sessionID)
			if err != nil {
				outputCh <- StreamEvent{Type: EventError, Data: err.Error()}
				return
			}
		}

		// 2. Append the new user prompt in-memory
		userCreatedAt := time.Now()
		if userPrompt != "" {
			history = append(history, database.Message{
				ConversationID: sessionID,
				Role:           "user",
				Content:        userPrompt,
				CreatedAt:      pgtype.Timestamptz{Time: userCreatedAt, Valid: true},
			})
		}

		// 3. Persist the user message in parallel with the LLM call.
		if s.queries != nil && userPrompt != "" {
			go func() {
				if err := s.queries.SessionAddChatHistory(ctx, database.SessionAddChatHistoryParams{
					ID:             uuid.New(),
					ConversationID: sessionID,
					Role:           "user",
					Content:        userPrompt,
					Metadata:       []database.Metadata{},
				}); err != nil {
					log.Printf("llm: save user message: %v", err)
				}
			}()
		}

		var turnText strings.Builder
		toolsMap := make(map[string]*database.Tool)

		// 4. Call CallLLM to handle the provider interactions and tool loop.
		streamCh, err := s.CallLLM(ctx, history)
		if err != nil {
			outputCh <- StreamEvent{Type: EventError, Data: err.Error()}
			return
		}

		// 5. Relay events, collect text, and intercept tool metadata for database saving.
		for event := range streamCh {
			switch event.Type {
			case EventText:
				turnText.WriteString(event.Content)
				outputCh <- event // forward to SSE immediately

			case EventToolCall:
				tc := event.Data.(database.Tool)
				if _, ok := toolsMap[tc.ID]; !ok {
					toolsMap[tc.ID] = &database.Tool{ID: tc.ID}
				}
				toolsMap[tc.ID].Name = tc.Name
				toolsMap[tc.ID].Arguments = tc.Arguments
				outputCh <- event

			case EventToolResult:
				tr := event.Data.(database.Tool)
				if _, ok := toolsMap[tr.ID]; !ok {
					toolsMap[tr.ID] = &database.Tool{ID: tr.ID}
				}
				toolsMap[tr.ID].Output = tr.Output
				toolsMap[tr.ID].IsError = tr.IsError
				outputCh <- event

			case EventDone:
				outputCh <- event

				var tools []database.Tool
				for _, t := range toolsMap {
					tools = append(tools, *t)
				}

				var meta []database.Metadata
				if len(tools) > 0 {
					meta = []database.Metadata{{Tool: tools}}
				} else {
					meta = []database.Metadata{}
				}

				body := turnText.String()
				if s.queries != nil && (body != "" || len(meta) > 0) {
					if err := s.queries.SessionAddChatHistory(ctx, database.SessionAddChatHistoryParams{
						ID:             uuid.New(),
						ConversationID: sessionID,
						Role:           string("assistant"),
						Content:        body,
						Metadata:       meta,
					}); err != nil {
						log.Printf("llm: save assistant message: %v", err)
					}
				}
				return

			case EventError:
				outputCh <- event
				return
			}
		}
	}()

	return outputCh, nil
}

// CallLLM provides a direct chat interface without any database interactions or session management.
func (s *LLMService) CallLLM(ctx context.Context, history []database.Message) (<-chan StreamEvent, error) {
	// Buffer nhỏ: provider vẫn có chút chỗ thở; relay Chat → SSE là unbuffered.
	outputCh := make(chan StreamEvent, 4)

	go func() {
		defer close(outputCh)

		toolDefs := s.tools.GetDefinitions()

	nextProviderRound:
		for {
			streamCh, err := s.completion(ctx, history, toolDefs)
			if err != nil {
				outputCh <- StreamEvent{Type: EventError, Data: err.Error()}
				return
			}

			var pendingTools []database.Tool

			for event := range streamCh {
				switch event.Type {
				case EventText:
					outputCh <- event

				case EventToolCall:
					tc := event.Data.(database.Tool)
					outputCh <- event
					pendingTools = append(pendingTools, tc)

				case EventDone:
					if len(pendingTools) > 0 {
						history = s.runToolRound(ctx, outputCh, history, pendingTools)
						pendingTools = pendingTools[:0]
						goto nextProviderRound
					}
					outputCh <- event
					return

				case EventError:
					outputCh <- event
					return
				}
			}

			if len(pendingTools) > 0 {
				history = s.runToolRound(ctx, outputCh, history, pendingTools)
				continue
			}
			return
		}
	}()

	return outputCh, nil
}
