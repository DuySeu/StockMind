package models

import (
	"encoding/json"
	"strings"
	"testing"

	"stockmind/internal/database"
)

// storedRound is an assistant row as it comes back from the database: one tool
// call whose result was stored unwrapped.
var storedRound = []database.Message{
	{Role: "user", Content: "giá VNM?"},
	{Role: "assistant", Metadata: []database.Metadata{{Tool: []database.Tool{
		{ID: "t1", Name: "get_stock_price", Arguments: "{}", Result: `{"close":61000}`},
	}}}},
}

const wantWrapped = "<tool_output source=\"get_stock_price\">\n{\"close\":61000}\n</tool_output>"

func TestConvertersWrapToolResults(t *testing.T) {
	t.Parallel()
	converters := map[string]func() (any, error){
		"anthropic":  func() (any, error) { return DBToAnthropicMessages(storedRound) },
		"openai":     func() (any, error) { return DBToOpenAIMessages(storedRound) },
		"openrouter": func() (any, error) { return DBToOpenRouterMessages(storedRound) },
	}
	for name, convert := range converters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			msgs, err := convert()
			if err != nil {
				t.Fatalf("convert: %v", err)
			}
			// The SDK types differ; their JSON encoding is what goes on the wire.
			raw, err := json.Marshal(msgs)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			want, _ := json.Marshal(wantWrapped) // a string always marshals
			if !strings.Contains(string(raw), strings.Trim(string(want), `"`)) {
				t.Errorf("%s request does not carry the wrapped tool result:\n%s", name, raw)
			}
		})
	}
}
