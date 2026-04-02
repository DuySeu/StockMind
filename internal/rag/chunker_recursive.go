package rag

import (
	"context"
	"errors"
	"strings"
)

// separators defines the ordered list of delimiters the recursive splitter
// tries when a chunk exceeds the target size. Higher-level separators are
// preferred so that semantic units (paragraphs, sentences) are kept intact.
var separators = []string{"\n\n", "\n", " ", ""}

// RecursiveChunker implements the Chunker interface using hierarchical
// character-based splitting. It is the default strategy for most document
// types because it balances chunk size with context preservation.
//
// Algorithm:
//  1. Try to split on "\n\n" (paragraph breaks).
//  2. If a segment is still too large, split on "\n" (line breaks).
//  3. If still too large, split on " " (words).
//  4. As a last resort, split on "" (any character).
//
// Overlap is added between consecutive chunks to avoid losing context at
// boundaries. The default target is 512 characters with ~10% (51-char) overlap.
type RecursiveChunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewRecursiveChunker creates a RecursiveChunker with the given size and overlap.
// If size <= 0, it defaults to 512. If overlap < 0, it defaults to 51 (≈10%).
func NewRecursiveChunker(size, overlap int) *RecursiveChunker {
	if size <= 0 {
		size = 512
	}
	if overlap < 0 {
		overlap = 51
	}
	return &RecursiveChunker{chunkSize: size, chunkOverlap: overlap}
}

// Strategy returns StrategyRecursive.
func (c *RecursiveChunker) Strategy() Strategy { return StrategyRecursive }

// Chunk splits text using hierarchical separators and returns overlapping
// segments of at most chunkSize characters.
func (c *RecursiveChunker) Chunk(_ context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}
	splits := splitRecursive(text, separators, c.chunkSize)
	return mergeWithOverlap(splits, c.chunkSize, c.chunkOverlap), nil
}

// splitRecursive recursively splits text until all pieces fit within maxSize
// using the ordered list of separators.
func splitRecursive(text string, seps []string, maxSize int) []string {
	if len(text) <= maxSize || len(seps) == 0 {
		if text == "" {
			return nil
		}
		return []string{text}
	}

	sep := seps[0]
	nextSeps := seps[1:]

	var parts []string
	if sep == "" {
		// Character-by-character split
		for i := 0; i < len(text); i += maxSize {
			end := i + maxSize
			if end > len(text) {
				end = len(text)
			}
			parts = append(parts, text[i:end])
		}
		return parts
	}

	segments := strings.Split(text, sep)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if len(seg) <= maxSize {
			parts = append(parts, seg)
		} else {
			parts = append(parts, splitRecursive(seg, nextSeps, maxSize)...)
		}
	}
	return parts
}

// mergeWithOverlap joins small splits back up to maxSize and adds overlap
// between consecutive chunks so context at boundaries is not lost.
func mergeWithOverlap(splits []string, maxSize, overlap int) []string {
	if len(splits) == 0 {
		return nil
	}

	var chunks []string
	var current strings.Builder

	for _, s := range splits {
		// If adding this piece exceeds the limit, flush current chunk.
		if current.Len() > 0 && current.Len()+1+len(s) > maxSize {
			chunk := strings.TrimSpace(current.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			// Seed next chunk with the tail of the previous for overlap.
			tail := current.String()
			if len(tail) > overlap {
				tail = tail[len(tail)-overlap:]
			}
			current.Reset()
			current.WriteString(tail)
		}

		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(s)
	}

	// Flush the final chunk.
	if chunk := strings.TrimSpace(current.String()); chunk != "" {
		chunks = append(chunks, chunk)
	}
	return chunks
}
