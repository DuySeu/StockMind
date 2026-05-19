package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"stockmind/internal/database"
	"stockmind/internal/llm/prompts"

	"github.com/google/uuid"
)

const summarizationThreshold int32 = 6

// triggerAsyncSummarization fires a background goroutine to summarize the conversation
// if the total message count has crossed a new 20-message threshold.
func (s *LLMService) triggerAsyncSummarization(sessionID uuid.UUID, totalCount int64, current database.ConversationSummary) {
	if totalCount < int64(current.SummarizedCount)+int64(summarizationThreshold) {
		return
	}

	go func() {
		// Use a background context — the request context may already be cancelled.
		ctx := context.Background()

		newThreshold := current.SummarizedCount + int64(summarizationThreshold)

		// Fetch the batch of messages to summarize (from last summarized point to new threshold).
		params := database.GetMessagesByConversationIDParams{
			ConversationID: sessionID,
			Limit:          summarizationThreshold,
			Offset:         int32(current.SummarizedCount),
		}
		batch, err := s.queries.GetMessagesByConversationID(ctx, params)
		if err != nil {
			log.Printf("summarizer: fetch batch: %v", err)
			return
		}

		// Build the messages text for the prompt.
		var msgBuf strings.Builder
		for _, m := range batch {
			fmt.Fprintf(&msgBuf, "%s: %s\n", m.Role, m.Content)
		}

		// Format existing key facts as a bulleted list.
		var factsStr string
		if len(current.KeyFacts) > 0 {
			factsStr = "- " + strings.Join(current.KeyFacts, "\n- ")
		}

		loader := prompts.NewPromptLoader()
		prompt, err := loader.GetSummarizationPrompt(prompts.SummarizationParams{
			Summary:  current.Summary,
			KeyFacts: factsStr,
			Messages: msgBuf.String(),
		})

		history := []database.Message{{Role: "user", Content: prompt}}
		streamCh, err := s.completion(ctx, history, nil)
		if err != nil {
			log.Printf("summarizer: llm call: %v", err)
			return
		}

		var sb strings.Builder
		for event := range streamCh {
			if event.Type == database.EventText {
				sb.WriteString(event.Content)
			}
		}

		// Parse the JSON response.
		raw := sb.String()
		// Strip markdown code fences if present.
		raw = strings.TrimSpace(raw)
		if strings.HasPrefix(raw, "```") {
			raw = raw[strings.Index(raw, "\n")+1:]
			raw = raw[:strings.LastIndex(raw, "```")]
			raw = strings.TrimSpace(raw)
		}

		var result struct {
			Summary  string   `json:"summary"`
			KeyFacts []string `json:"key_facts"`
		}
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			log.Printf("summarizer: parse response: %v (raw: %s)", err, raw)
			return
		}

		updated := database.ConversationSummary{
			Summary:         result.Summary,
			KeyFacts:        result.KeyFacts,
			SummarizedCount: newThreshold,
		}
		metaBytes, err := json.Marshal(updated)
		if err != nil {
			log.Printf("summarizer: marshal metadata: %v", err)
			return
		}

		if err := s.queries.UpdateConversationMetadata(ctx, database.UpdateConversationMetadataParams{
			ID:       sessionID,
			Metadata: metaBytes,
		}); err != nil {
			log.Printf("summarizer: update metadata: %v", err)
		}
	}()
}
