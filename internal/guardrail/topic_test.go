package guardrail

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"stockmind/internal/database"
)

// fakeLLM answers every structured completion with reply, or fails with err.
// It records the last prompt so tests can check what context was sent.
type fakeLLM struct {
	reply  string
	err    error
	delay  time.Duration
	prompt string
}

func (f *fakeLLM) StructuredCompletion(ctx context.Context, prompt string, result any) error {
	f.prompt = prompt
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal([]byte(f.reply), result)
}

func TestClassify(t *testing.T) {
	t.Parallel()
	question := []database.Message{{Role: "user", Content: "hướng dẫn nấu phở"}}
	tests := []struct {
		name        string
		llm         *fakeLLM
		wantAllowed bool
		wantErr     bool
	}{
		{"allows_in_scope_message", &fakeLLM{reply: `{"allowed":true,"category":"stock","reason":"ticker"}`}, true, false},
		{"blocks_off_topic_message", &fakeLLM{reply: `{"allowed":false,"category":"off_topic","reason":"cooking"}`}, false, false},
		{"fails_open_on_llm_error", &fakeLLM{err: errors.New("quota exceeded")}, true, true},
		{"fails_open_when_allowed_missing", &fakeLLM{reply: `{"category":"off_topic"}`}, true, true},
		{"fails_open_on_unknown_category", &fakeLLM{reply: `{"allowed":false,"category":"cooking"}`}, true, true},
		{"fails_open_on_timeout", &fakeLLM{reply: `{"allowed":false,"category":"off_topic"}`, delay: time.Second}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewClassifier(tt.llm, 50*time.Millisecond)
			got, err := c.Classify(t.Context(), question)
			if got.Allowed != tt.wantAllowed || (err != nil) != tt.wantErr {
				t.Errorf("Classify = %+v, err %v; want allowed %v, err %v", got, err, tt.wantAllowed, tt.wantErr)
			}
		})
	}
}

func TestClassifySkipsRefusedTurnsInContext(t *testing.T) {
	t.Parallel()
	llm := &fakeLLM{reply: `{"allowed":true,"category":"stock","reason":"follow-up"}`}
	blocked := []database.Metadata{{Guardrail: &database.GuardrailInfo{Action: ActionBlocked}}}
	history := []database.Message{
		{Role: "user", Content: "giá VNM hôm nay?"},
		{Role: "assistant", Content: "VNM đóng cửa 61.000."},
		{Role: "user", Content: "viết giúp tôi bài thơ"},
		{Role: "assistant", Content: Refusal, Metadata: blocked},
		{Role: "user", Content: "còn FPT thì sao?"},
	}

	if _, err := NewClassifier(llm, time.Second).Classify(t.Context(), history); err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if strings.Contains(llm.prompt, "bài thơ") || strings.Contains(llm.prompt, Refusal) {
		t.Errorf("prompt kept the refused turn:\n%s", llm.prompt)
	}
	for _, want := range []string{"user: giá VNM hôm nay?", "assistant: VNM đóng cửa 61.000.", "còn FPT thì sao?"} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("prompt is missing %q:\n%s", want, llm.prompt)
		}
	}
}
