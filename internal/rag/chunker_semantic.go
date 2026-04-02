package rag

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode"
)

// SemanticChunker splits text at points where the topic shifts, as detected
// by a drop in cosine similarity between adjacent sentence embeddings.
//
// Unlike character-based chunkers, SemanticChunker requires access to an
// Embedder so that it can generate vector representations on the fly. This
// makes it more expensive but produces boundaries that are semantically
// meaningful rather than structurally arbitrary.
//
// Algorithm:
//  1. Tokenise the text into sentences.
//  2. Embed each sentence using the provided Embedder.
//  3. Compute cosine similarity between each adjacent pair.
//  4. Where similarity drops below `threshold`, start a new chunk.
//
// Default threshold: 0.70 (matching the RAG retrieval threshold).
type SemanticChunker struct {
	embedder  Embedder
	threshold float64
}

// NewSemanticChunker creates a SemanticChunker backed by the given Embedder.
// If threshold <= 0, it defaults to 0.70.
func NewSemanticChunker(embedder Embedder, threshold float64) *SemanticChunker {
	if threshold <= 0 {
		threshold = 0.70
	}
	return &SemanticChunker{embedder: embedder, threshold: threshold}
}

// Strategy returns StrategySemantic.
func (c *SemanticChunker) Strategy() Strategy { return StrategySemantic }

// Chunk splits text at semantic boundaries detected via embedding similarity.
func (c *SemanticChunker) Chunk(ctx context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}
	if c.embedder == nil {
		return nil, errors.New("chunker: embedder must not be nil for semantic chunking")
	}

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil, errors.New("chunker: no sentences found in text")
	}
	if len(sentences) == 1 {
		return sentences, nil
	}

	// Embed all sentences in one call (embedder batches internally).
	embeddings, err := c.embedder.Embed(ctx, sentences)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != len(sentences) {
		return nil, errors.New("chunker: embedding count does not match sentence count")
	}

	// Walk adjacent pairs and group into chunks.
	var chunks []string
	start := 0
	for i := 1; i < len(sentences); i++ {
		sim := cosineSimilarity(embeddings[i-1], embeddings[i])
		if sim < c.threshold {
			// Topic shift detected — flush the current group.
			chunks = append(chunks, strings.Join(sentences[start:i], " "))
			start = i
		}
	}
	// Flush the trailing group.
	if start < len(sentences) {
		chunks = append(chunks, strings.Join(sentences[start:], " "))
	}
	return chunks, nil
}

// splitSentences is a lightweight sentence tokeniser that splits on common
// terminal punctuation followed by whitespace or end-of-string.
// For Vietnamese and English documents this is sufficient without a full NLP library.
func splitSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var sentences []string
	var buf strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		buf.WriteRune(r)

		isTerminal := r == '.' || r == '!' || r == '?' || r == '…'
		if !isTerminal {
			continue
		}

		// Peek ahead — only split if the next non-trivial character is
		// uppercase or the string is exhausted.
		next := i + 1
		for next < len(runes) && unicode.IsSpace(runes[next]) {
			next++
		}
		atEnd := next >= len(runes)
		nextIsUpper := atEnd || unicode.IsUpper(runes[next])

		if atEnd || nextIsUpper {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			buf.Reset()
		}
	}
	// Flush any remaining text (no terminal punctuation).
	if s := strings.TrimSpace(buf.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
