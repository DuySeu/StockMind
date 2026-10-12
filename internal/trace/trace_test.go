package trace

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestStart(t *testing.T) {
	t.Run("start without recorder is noop", func(t *testing.T) {
		ctx := context.Background()
		got, end := Start(ctx, "tool", "tool.x", "in")
		end("out", nil)
		Annotate(got, "k", "v")
		if got != ctx {
			t.Errorf("Start without recorder returned a new context; want the same one")
		}
	})

	t.Run("child span gets parent id from ctx", func(t *testing.T) {
		rec := NewRecorder(16)
		ctx := WithRecorder(context.Background(), rec)
		parentCtx, endParent := Start(ctx, "handler", "chat.turn", nil)
		_, endChild := Start(parentCtx, "tool", "tool.x", nil)
		endChild(nil, nil)
		endParent(nil, nil)

		events := rec.Drain()
		if len(events) != 4 {
			t.Fatalf("got %d events; want 4", len(events))
		}
		parent, child := events[0].Span, events[1].Span
		if child.ParentID != parent.ID || parent.ParentID != "" {
			t.Errorf("child.ParentID = %q, parent.ID = %q, parent.ParentID = %q", child.ParentID, parent.ID, parent.ParentID)
		}
	})

	t.Run("end records output and error once", func(t *testing.T) {
		rec := NewRecorder(16)
		_, end := Start(WithRecorder(context.Background(), rec), "tool", "tool.x", nil)
		end("first", errors.New("boom"))
		end("second", nil)

		events := rec.Drain()
		if len(events) != 2 {
			t.Fatalf("got %d events; want 2", len(events))
		}
		span := events[1].Span
		if events[1].Phase != PhaseEnd || span.Error != "boom" || string(span.Output.(json.RawMessage)) != `"first"` {
			t.Errorf("end event = %+v; want phase end, error boom, output \"first\"", span)
		}
	})

	t.Run("drops span when buffer full without blocking", func(t *testing.T) {
		rec := NewRecorder(1)
		ctx := WithRecorder(context.Background(), rec)
		_, end := Start(ctx, "tool", "tool.x", nil)
		end(nil, nil)
		if got := rec.Dropped(); got != 1 {
			t.Errorf("Dropped() = %d; want 1", got)
		}
	})

	t.Run("truncates large output", func(t *testing.T) {
		rec := NewRecorder(16)
		_, end := Start(WithRecorder(context.Background(), rec), "tool", "tool.x", nil)
		end(strings.Repeat("a", MaxPayload*2), nil)

		span := rec.Drain()[1].Span
		out, ok := span.Output.(string)
		if !ok || len(out) != MaxPayload || !span.Truncated {
			t.Errorf("Output len = %d, Truncated = %v; want %d and true", len(out), span.Truncated, MaxPayload)
		}
	})

	t.Run("annotate adds attr to current span", func(t *testing.T) {
		rec := NewRecorder(16)
		ctx, end := Start(WithRecorder(context.Background(), rec), "guardrail", "guardrail.classify", nil)
		Annotate(ctx, "category", "stock")
		end(nil, nil)

		span := rec.Drain()[1].Span
		if span.Attrs["category"] != "stock" {
			t.Errorf("Attrs = %v; want category=stock", span.Attrs)
		}
	})

	t.Run("span start event is not mutated by later annotate", func(t *testing.T) {
		rec := NewRecorder(16)
		ctx, end := Start(WithRecorder(context.Background(), rec), "llm", "llm.round#1", nil)
		Annotate(ctx, "ttft_ms", 5)
		end(nil, nil)

		start := rec.Drain()[0].Span
		if len(start.Attrs) != 0 {
			t.Errorf("start event Attrs = %v; want empty", start.Attrs)
		}
	})
}
