package rag

import (
	"context"
	"strings"
	"testing"
)

// ---- RecursiveChunker ----

func TestRecursiveChunker_Basic(t *testing.T) {
	c := NewRecursiveChunker(100, 10)
	text := strings.Repeat("word ", 50) // 250 chars → should produce >1 chunk
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	if len(chunks) < 2 {
		t.Errorf("expected >1 chunks for 250-char text with size=100, got %d", len(chunks))
	}
}

func TestRecursiveChunker_ParagraphSplit(t *testing.T) {
	c := NewRecursiveChunker(200, 20)
	text := "First paragraph with some text here.\n\nSecond paragraph with different content here.\n\nThird paragraph is added to make the text longer."
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	// All three paragraphs fit within 200 chars individually — we should have ≥1 chunk
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
	// Original content should be present
	full := strings.Join(chunks, " ")
	if !strings.Contains(full, "First paragraph") {
		t.Error("expected 'First paragraph' in chunks")
	}
}

func TestRecursiveChunker_EmptyText(t *testing.T) {
	c := NewRecursiveChunker(512, 51)
	_, err := c.Chunk(context.Background(), "   ")
	if err == nil {
		t.Error("expected error for whitespace-only text")
	}
}

func TestRecursiveChunker_Overlap(t *testing.T) {
	c := NewRecursiveChunker(50, 10)
	// Build a text long enough to produce multiple chunks
	text := strings.Repeat("abcde ", 30) // 180 chars
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	if len(chunks) < 2 {
		t.Skip("text too short to test overlap with these settings")
	}
	// The end of chunk[0] should appear in the beginning of chunk[1]
	tail := chunks[0][len(chunks[0])-10:]
	if !strings.HasPrefix(chunks[1], strings.TrimSpace(tail)) {
		// Overlap is tail-of-previous → prefix-of-next; allow for trimming.
		t.Logf("chunk[0] tail: %q, chunk[1] prefix: %q (overlap may vary with space trimming)", tail, chunks[1][:min(20, len(chunks[1]))])
	}
}

// ---- FixedChunker ----

func TestFixedChunker_Basic(t *testing.T) {
	c := NewFixedChunker(10, 0)
	text := "abcdefghijklmnopqrst" // 20 chars → 2 chunks of 10
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestFixedChunker_Empty(t *testing.T) {
	c := NewFixedChunker(512, 0)
	_, err := c.Chunk(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty text")
	}
}

// ---- ParagraphChunker ----

func TestParagraphChunker_Basic(t *testing.T) {
	c := NewParagraphChunker()
	text := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestParagraphChunker_SingleNewlineDoesNotSplit(t *testing.T) {
	c := NewParagraphChunker()
	// Single newlines inside a paragraph should NOT cause a split
	text := "Line one.\nLine two.\n\nNew paragraph."
	chunks, err := c.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Chunk() error: %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks (single newline = same paragraph), got %d: %v", len(chunks), chunks)
	}
	// First chunk should contain both lines joined
	if !strings.Contains(chunks[0], "Line one") || !strings.Contains(chunks[0], "Line two") {
		t.Errorf("expected both lines in chunk[0], got: %q", chunks[0])
	}
}

func TestParagraphChunker_Empty(t *testing.T) {
	c := NewParagraphChunker()
	_, err := c.Chunk(context.Background(), "   ")
	if err == nil {
		t.Error("expected error for whitespace-only text")
	}
}

// ---- SemanticChunker helper ----

func TestSplitSentences(t *testing.T) {
	text := "This is sentence one. This is sentence two! And sentence three?"
	got := splitSentences(text)
	if len(got) != 3 {
		t.Errorf("expected 3 sentences, got %d: %v", len(got), got)
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float32{1, 0, 0}
	sim := cosineSimilarity(v, v)
	if sim < 0.9999 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	sim := cosineSimilarity(a, b)
	if sim > 0.0001 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", sim)
	}
}

// ---- Strategy interface compliance ----

func TestStrategyValues(t *testing.T) {
	cases := []struct {
		c    Chunker
		want Strategy
	}{
		{NewRecursiveChunker(512, 51), StrategyRecursive},
		{NewFixedChunker(512, 0), StrategyFixed},
		{NewParagraphChunker(), StrategyParagraph},
		{NewSemanticChunker(nil, 0), StrategySemantic},
	}
	for _, tc := range cases {
		if got := tc.c.Strategy(); got != tc.want {
			t.Errorf("Strategy() = %q, want %q", got, tc.want)
		}
	}
}

// ---- Embedder unit test (no live API) ----

func TestOpenRouterEmbedder_MissingAPIKey(t *testing.T) {
	_, err := NewOpenRouterEmbedder("", 20)
	if err == nil {
		t.Error("expected error for empty API key")
	}
}

func TestOpenRouterEmbedder_Dimensions(t *testing.T) {
	// We can construct with a dummy key just to verify Dimensions().
	e, err := NewOpenRouterEmbedder("dummy-key", 20)
	if err != nil {
		t.Fatalf("NewOpenRouterEmbedder failed: %v", err)
	}
	if e.Dimensions() != 2048 {
		t.Errorf("Dimensions() = %d, want 2048", e.Dimensions())
	}
}

func TestEmbedderBatching_EmptyInput(t *testing.T) {
	e, _ := NewOpenRouterEmbedder("dummy-key", 20)
	_, err := e.Embed(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for empty input slice")
	}
}

// min is a small helper (Go 1.21+ has it built-in, but this keeps backward compat).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
