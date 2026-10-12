// Package trace records a tree of timed spans for one request, for the eval
// page. A context without a Recorder makes every call a no-op, so production
// requests pay nothing.
package trace

import (
	"context"
	"encoding/json"
	"maps"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// MaxPayload caps each serialized Input and Output: a system prompt or a RAG
// result would otherwise put hundreds of KB on the wire per span.
const MaxPayload = 8 * 1024

// Event phases sent to the client.
const (
	PhaseStart   = "start"
	PhaseEnd     = "end"
	PhaseSummary = "summary"
)

type ctxKey int

const (
	recorderKey ctxKey = iota
	spanKey
)

// Span is one timed step. Input and Output hold JSON, or a cut-off string when
// the JSON was longer than MaxPayload.
type Span struct {
	ID         string         `json:"id"`
	ParentID   string         `json:"parent_id,omitempty"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Start      time.Time      `json:"start"`
	DurationMs int64          `json:"duration_ms"`
	Input      any            `json:"input,omitempty"`
	Output     any            `json:"output,omitempty"`
	Attrs      map[string]any `json:"attrs,omitempty"`
	Error      string         `json:"error,omitempty"`
	Truncated  bool           `json:"truncated,omitempty"`
}

// Event is one message to the client: a span starting or ending, or the final
// summary of dropped spans.
type Event struct {
	Phase   string `json:"phase"`
	Span    *Span  `json:"span,omitempty"`
	Dropped int64  `json:"dropped,omitempty"`
}

// Recorder buffers span events for one request. Sends never block: a full
// buffer drops the event and counts it.
type Recorder struct {
	events  chan Event
	nextID  atomic.Int64
	dropped atomic.Int64
}

// active is a span still running. Producers in several goroutines may annotate
// it, so every access goes through mu.
type active struct {
	mu   sync.Mutex
	span Span
	once sync.Once
	rec  *Recorder
}

// NewRecorder buffers up to size events between drains.
func NewRecorder(size int) *Recorder {
	return &Recorder{events: make(chan Event, size)}
}

// Events is the channel a relay loop reads.
func (r *Recorder) Events() <-chan Event { return r.events }

// Drain returns every event buffered so far without waiting for more.
func (r *Recorder) Drain() []Event {
	var out []Event
	for {
		select {
		case ev := <-r.events:
			out = append(out, ev)
		default:
			return out
		}
	}
}

// Dropped is how many events were lost to a full buffer.
func (r *Recorder) Dropped() int64 { return r.dropped.Load() }

// WithRecorder turns tracing on for everything run under ctx.
func WithRecorder(ctx context.Context, rec *Recorder) context.Context {
	return context.WithValue(ctx, recorderKey, rec)
}

// Start opens a child of the span in ctx and returns a context carrying it plus
// the function that closes it. end is safe to call more than once; only the
// first call counts.
func Start(ctx context.Context, kind, name string, input any) (context.Context, func(output any, err error)) {
	rec, _ := ctx.Value(recorderKey).(*Recorder)
	if rec == nil {
		return ctx, func(any, error) {}
	}

	// Build the span under its parent
	a := &active{rec: rec, span: Span{
		ID:    strconv.FormatInt(rec.nextID.Add(1), 10),
		Kind:  kind,
		Name:  name,
		Start: time.Now(),
	}}
	if parent, ok := ctx.Value(spanKey).(*active); ok {
		a.span.ParentID = parent.span.ID
	}
	a.span.Input = payload(input, &a.span.Truncated)
	a.emit(PhaseStart)

	end := func(output any, err error) {
		a.once.Do(func() {
			a.mu.Lock()
			a.span.DurationMs = time.Since(a.span.Start).Milliseconds()
			a.span.Output = payload(output, &a.span.Truncated)
			if err != nil {
				a.span.Error = err.Error()
			}
			a.mu.Unlock()
			a.emit(PhaseEnd)
		})
	}
	return context.WithValue(ctx, spanKey, a), end
}

// Annotate sets one attribute on the span in ctx.
func Annotate(ctx context.Context, key string, value any) {
	a, ok := ctx.Value(spanKey).(*active)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.span.Attrs == nil {
		a.span.Attrs = make(map[string]any)
	}
	a.span.Attrs[key] = value
}

// emit sends a copy of the span, so later writes never race the reader.
func (a *active) emit(phase string) {
	a.mu.Lock()
	snapshot := a.span
	snapshot.Attrs = maps.Clone(a.span.Attrs)
	a.mu.Unlock()

	select {
	case a.rec.events <- Event{Phase: phase, Span: &snapshot}:
	default:
		a.rec.dropped.Add(1)
	}
}

// payload serializes v once, so the client sees the value as it was at this
// moment even if the caller reuses it afterwards.
func payload(v any, truncated *bool) any {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "unserializable: " + err.Error()
	}
	if len(data) > MaxPayload {
		*truncated = true
		return string(data[:MaxPayload])
	}
	return json.RawMessage(data)
}
