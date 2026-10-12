//go:build integration

package guardrail_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/guardrail"
	"stockmind/internal/llm/models"

	"github.com/joho/godotenv"
)

// TestClassifierAgainstLabelledTopics runs the real guard model over
// testdata/topics.json and reports its accuracy; each miss fails by message.
func TestClassifierAgainstLabelledTopics(t *testing.T) {
	if testing.Short() {
		t.Skip("calls a live model")
	}
	_ = godotenv.Load("../../.env") // optional: env may already be set
	cfg := common.LoadConfig()
	model := cfg.Guardrail.Model
	if model == "" {
		model = cfg.LLM.GetLLMModelName()
	}
	if os.Getenv("OPENROUTER_API_KEY") == "" || model == "" {
		t.Skip("OPENROUTER_API_KEY and GUARD_MODEL or LLM_MODEL are required")
	}

	provider, err := models.NewProviderForModel(t.Context(), cfg.LLM, model)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	classifier := guardrail.NewClassifier(provider, 30*time.Second)

	raw, err := os.ReadFile("testdata/topics.json")
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}
	var cases []struct {
		Message string             `json:"message"`
		History []database.Message `json:"history"`
		Allowed bool               `json:"allowed"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse cases: %v", err)
	}

	correct := 0
	for _, c := range cases {
		history := append(c.History, database.Message{Role: "user", Content: c.Message})
		got, err := classifier.Classify(t.Context(), history)
		if err != nil {
			t.Errorf("%q: classify failed: %v", c.Message, err)
			continue
		}
		if got.Allowed != c.Allowed {
			t.Errorf("%q: allowed = %v (%s: %s); want %v", c.Message, got.Allowed, got.Category, got.Reason, c.Allowed)
			continue
		}
		correct++
	}
	t.Logf("model %s: %d/%d correct", model, correct, len(cases))
}
