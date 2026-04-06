package rag

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode"
)

// Strategy defines the chunking algorithm to use when splitting text.
type Strategy string

const (
	// StrategyRecursive splits text hierarchically using structural separators.
	// Preferred default: balances context preservation with chunk size.
	StrategyRecursive Strategy = "recursive"

	// StrategyFixed splits text at exact character-count boundaries.
	StrategyFixed Strategy = "fixed"

	// StrategyParagraph splits text at paragraph boundaries (\n\n).
	StrategyParagraph Strategy = "paragraph"

	// StrategySemantic splits text based on embedding cosine-similarity shifts.
	StrategySemantic Strategy = "semantic"
)

// Chunker splits a raw text document into smaller overlapping segments
// suitable for embedding and vector storage.
type Chunker interface {
	// Chunk splits text into segments. Returns an error if the text cannot
	// be processed (e.g., nil or empty).
	Chunk(ctx context.Context, text string) ([]string, error)

	// Strategy returns the splitting algorithm this Chunker uses.
	Strategy() Strategy
}

// ---- Recursive Chunker ----

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

// ---- Fixed Chunker ----

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

// ---- Paragraph Chunker ----

// ParagraphChunker splits text at paragraph boundaries (double newlines).
// Each paragraph is returned as an individual chunk. Consecutive lines
// without a blank line between them are treated as a single paragraph chunk.
//
// Best suited for documents with well-defined paragraphs such as Markdown
// reports and DOCX bodies where section breaks carry semantic meaning.
type ParagraphChunker struct{}

// NewParagraphChunker returns a new ParagraphChunker.
func NewParagraphChunker() *ParagraphChunker { return &ParagraphChunker{} }

// Strategy returns StrategyParagraph.
func (c *ParagraphChunker) Strategy() Strategy { return StrategyParagraph }

// Chunk splits text by blank-line boundaries. Single newlines inside a
// paragraph do NOT cause a split — only double newlines ("\n\n") do.
func (c *ParagraphChunker) Chunk(_ context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}

	// Normalise Windows-style line endings first.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	raw := strings.Split(text, "\n\n")
	var chunks []string
	for _, para := range raw {
		// Collapse internal single newlines to spaces to keep the chunk
		// readable as a single block.
		para = strings.ReplaceAll(para, "\n", " ")
		para = strings.TrimSpace(para)
		if para != "" {
			chunks = append(chunks, para)
		}
	}
	return chunks, nil
}

// ---- Semantic Chunker ----

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
