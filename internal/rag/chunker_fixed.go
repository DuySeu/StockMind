package rag

import (
	"context"
	"errors"
	"strings"
)

// FixedChunker splits text into segments of exactly chunkSize characters
// with a configurable overlap. Use when you need predictable chunk sizes for
// storage estimation or when document structure is irrelevant.
type FixedChunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewFixedChunker creates a FixedChunker. size must be > 0; overlap must be
// >= 0 and < size.
func NewFixedChunker(size, overlap int) *FixedChunker {
	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	return &FixedChunker{chunkSize: size, chunkOverlap: overlap}
}

// Strategy returns StrategyFixed.
func (c *FixedChunker) Strategy() Strategy { return StrategyFixed }

// Chunk splits text into fixed-size windows with overlap.
func (c *FixedChunker) Chunk(_ context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}

	step := c.chunkSize - c.chunkOverlap
	if step <= 0 {
		step = c.chunkSize
	}

	var chunks []string
	for i := 0; i < len(text); i += step {
		end := i + c.chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunk := strings.TrimSpace(text[i:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}
