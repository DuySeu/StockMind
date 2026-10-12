package guardrail

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"stockmind/internal/database"
	"stockmind/internal/llm/prompts"
	"stockmind/internal/trace"
)

// Refusal is the fixed reply to an off-topic question; it costs no LLM call.
const Refusal = "Mình chỉ hỗ trợ các câu hỏi về chứng khoán và tài chính, ví dụ mã cổ phiếu, báo cáo tài chính, tin tức thị trường, vĩ mô hoặc kiến thức đầu tư. Bạn thử hỏi lại theo hướng đó nhé."

// ActionBlocked marks an assistant message that is a refusal.
const ActionBlocked = "blocked"

const (
	recentTurns    = 3
	recentTurnRune = 500
)

var categories = []string{"stock", "finance", "greeting", "app_help", "off_topic"}

// StructuredCompleter is the one LLM call the classifier needs.
type StructuredCompleter interface {
	StructuredCompletion(ctx context.Context, prompt string, result any) error
}

// Decision is the classifier's verdict on one message. Category and Reason are
// for logs only.
type Decision struct {
	Allowed  bool
	Category string
	Reason   string
}

// Classifier decides whether a chat message is within StockMind's scope.
type Classifier struct {
	llm     StructuredCompleter
	prompts *prompts.PromptLoader
	timeout time.Duration
}

// NewClassifier builds a classifier that gives each call at most timeout.
func NewClassifier(llm StructuredCompleter, timeout time.Duration) *Classifier {
	return &Classifier{llm: llm, prompts: prompts.NewPromptLoader(), timeout: timeout}
}

// Classify judges the last message of history, using the turns before it as
// context. It fails open: on any error the decision allows the message, and the
// error is returned for the caller to log.
func (c *Classifier) Classify(ctx context.Context, history []database.Message) (d Decision, err error) {
	allow := Decision{Allowed: true}
	if len(history) == 0 {
		return allow, nil
	}

	// Render the prompt from recent context plus the message itself
	last := history[len(history)-1]
	recent := renderRecent(history[:len(history)-1])
	ctx, end := trace.Start(ctx, "guardrail", "guardrail.classify", map[string]string{"history": recent, "message": last.Content})
	defer func() {
		trace.Annotate(ctx, "fail_open", err != nil)
		end(d, err)
	}()
	prompt, err := c.prompts.GetGuardTopicPrompt(prompts.GuardTopicParams{
		History: recent,
		Message: last.Content,
	})
	if err != nil {
		return allow, fmt.Errorf("classify: %w", err)
	}

	// Ask the model under the classifier's own deadline
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var raw struct {
		Allowed  *bool  `json:"allowed"`
		Category string `json:"category"`
		Reason   string `json:"reason"`
	}
	if err := c.llm.StructuredCompletion(ctx, prompt, &raw); err != nil {
		return allow, fmt.Errorf("classify: %w", err)
	}

	// Reject a verdict that does not follow the contract
	if raw.Allowed == nil {
		return allow, errors.New("classify: response has no allowed field")
	}
	if !slices.Contains(categories, raw.Category) {
		return allow, fmt.Errorf("classify: unknown category %q", raw.Category)
	}
	return Decision{Allowed: *raw.Allowed, Category: raw.Category, Reason: raw.Reason}, nil
}

// renderRecent writes the last few user/assistant turns as "role: text" lines.
// A refusal and the question it answered are skipped, so one blocked turn does
// not bias the next verdict.
func renderRecent(history []database.Message) string {
	var picked []string
	for i := len(history) - 1; i >= 0 && len(picked) < recentTurns; i-- {
		m := history[i]
		if m.Role == "assistant" && len(m.Metadata) > 0 && m.Metadata[0].Guardrail != nil {
			i-- // also skip the user message the refusal answered
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" || (m.Role != "user" && m.Role != "assistant") {
			continue
		}
		if r := []rune(content); len(r) > recentTurnRune {
			content = string(r[:recentTurnRune]) + "..."
		}
		picked = append(picked, m.Role+": "+content)
	}
	slices.Reverse(picked)
	return strings.Join(picked, "\n")
}
