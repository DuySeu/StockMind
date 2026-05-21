package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/providers"
	"stockmind/internal/llm/tools"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type completionFunc func(context.Context, []database.Message, []*tools.Tool) (<-chan database.StreamEvent, error)

type LLMService struct {
	queries    *database.Queries
	tools      *tools.Manager
	completion completionFunc
}

func NewLLMService(ctx context.Context, providerName database.ModelProvider, model string, cfg common.LLMProvider, pool *pgxpool.Pool, toolMgr *tools.Manager) (*LLMService, error) {
	var completion completionFunc
	switch providerName {
	case database.ModelProviderOpenAI:
		client, err := providers.NewOpenAIClient(cfg.OpenAI)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
			return providers.OpenAICompletion(client, model, ctx, history, tools)
		}
	case database.ModelProviderAnthropic:
		client, err := providers.NewAnthropicClient(ctx, cfg.Anthropic)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
			return providers.AnthropicCompletion(client, model, ctx, history, tools)
		}
	case database.ModelProviderOpenRouter:
		client, err := providers.NewOpenRouterClient(cfg.OpenRouter)
		if err != nil {
			return nil, err
		}
		completion = func(ctx context.Context, history []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
			return providers.OpenRouterCompletion(client, model, ctx, history, tools)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %q", providerName)
	}

	return &LLMService{
		queries:    database.New(pool),
		tools:      toolMgr,
		completion: completion,
	}, nil
}

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
		assembled = append(assembled, database.Tool{
			ID:      tc.ID,
			Result:  tr.Result,
			IsError: tr.IsError,
		})
	}
	meta := []database.Metadata{{Tool: assembled}}
	return append(history, database.Message{Role: "assistant", Metadata: meta})
}

func (s *LLMService) Chat(ctx context.Context, sessionID uuid.UUID, userPrompt string) (<-chan database.StreamEvent, error) {
	// Unbuffered: mỗi event LLM chỉ rời relay khi HTTP handler đã nhận (ít gom trong RAM).
	outputCh := make(chan database.StreamEvent)

	go func() {
		defer close(outputCh)

		// 1. Load conversation metadata (summary + key facts) and last 20 messages.
		var history []database.Message
		var convSummary database.ConversationSummary
		if s.queries != nil {
			conv, err := s.queries.GetConversationByID(ctx, sessionID)
			if err != nil {
				outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
				return
			}
			if len(conv.Metadata) > 0 {
				_ = json.Unmarshal(conv.Metadata, &convSummary)
			}

			history, err = s.queries.GetMessagesByConversationID(ctx, database.GetMessagesByConversationIDParams{
				ConversationID: sessionID,
				Limit:          20,
				Offset:         0,
			})
			if err != nil {
				outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
				return
			}
		}

		// 2. Prepend summary as a system message if one exists.
		if convSummary.Summary != "" {
			var sb strings.Builder
			sb.WriteString("Previous conversation context:\n")
			sb.WriteString(convSummary.Summary)
			if len(convSummary.KeyFacts) > 0 {
				sb.WriteString("\n\nKey facts from this conversation:\n- ")
				sb.WriteString(strings.Join(convSummary.KeyFacts, "\n- "))
			}
			history = append([]database.Message{{Role: "system", Content: sb.String()}}, history...)
		}

		// 3. Append the new user prompt in-memory.
		if userPrompt != "" {
			history = append(history, database.Message{
				ConversationID: sessionID,
				Role:           "user",
				Content:        userPrompt,
			})
		}

		// 4. Persist the user message in parallel with the LLM call.
		if s.queries != nil && userPrompt != "" {
			go func() {
				if err := s.queries.CreateMessage(ctx, database.CreateMessageParams{
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

		// 5. Call CallLLM to handle the provider interactions and tool loop.
		streamCh, err := s.CallLLM(ctx, history)
		if err != nil {
			outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
			return
		}

		// 6. Relay events, collect text, and intercept tool metadata for database saving.
		for event := range streamCh {
			switch event.Type {
			case database.EventText:
				turnText.WriteString(event.Content)
				outputCh <- event // forward to SSE immediately

			case database.EventToolCall:
				tc := event.Data.(database.Tool)
				if _, ok := toolsMap[tc.ID]; !ok {
					toolsMap[tc.ID] = &database.Tool{ID: tc.ID}
				}
				toolsMap[tc.ID].Name = tc.Name
				toolsMap[tc.ID].Arguments = tc.Arguments
				outputCh <- event

			case database.EventToolResult:
				tr := event.Data.(database.Tool)
				if _, ok := toolsMap[tr.ID]; !ok {
					toolsMap[tr.ID] = &database.Tool{ID: tr.ID}
				}
				toolsMap[tr.ID].Result = tr.Result
				toolsMap[tr.ID].IsError = tr.IsError
				outputCh <- event

			case database.EventDone:
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
					if err := s.queries.CreateMessage(ctx, database.CreateMessageParams{
						ID:             uuid.New(),
						ConversationID: sessionID,
						Role:           "assistant",
						Content:        body,
						Metadata:       meta,
					}); err != nil {
						log.Printf("llm: save assistant message: %v", err)
					}
				}

				// 7. Trigger async summarization if threshold crossed.
				if s.queries != nil {
					go func() {
						count, err := s.queries.GetMessageCountByConversationID(context.Background(), sessionID)
						if err != nil {
							log.Printf("llm: count messages: %v", err)
							return
						}
						s.triggerAsyncSummarization(sessionID, count, convSummary)
					}()
				}
				return

			case database.EventError:
				outputCh <- event
				return
			}
		}
	}()

	return outputCh, nil
}

// CallLLM provides a direct chat interface without any database interactions or session management.
func (s *LLMService) CallLLM(ctx context.Context, history []database.Message) (<-chan database.StreamEvent, error) {
	// Buffer nhỏ: provider vẫn có chút chỗ thở; relay Chat → SSE là unbuffered.
	outputCh := make(chan database.StreamEvent, 4)

	go func() {
		defer close(outputCh)

		toolDefs := s.tools.All()

	nextProviderRound:
		for {
			streamCh, err := s.completion(ctx, history, toolDefs)
			if err != nil {
				outputCh <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
				return
			}

			var pendingTools []database.Tool

			for event := range streamCh {
				switch event.Type {
				case database.EventText:
					outputCh <- event

				case database.EventToolCall:
					tc := event.Data.(database.Tool)
					outputCh <- event
					pendingTools = append(pendingTools, tc)

				case database.EventDone:
					if len(pendingTools) > 0 {
						history = s.runToolRound(ctx, outputCh, history, pendingTools)
						pendingTools = pendingTools[:0]
						goto nextProviderRound
					}
					outputCh <- event
					return

				case database.EventError:
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
