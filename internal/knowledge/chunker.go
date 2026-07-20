package knowledge

import (
	"context"
	"errors"
	"math"
	"stockmind/internal/database"
	"strings"
	"unicode"
)

// Chunker splits text into smaller segments for embedding.
type Chunker interface {
	Chunk(ctx context.Context, text string) ([]string, error)
	Strategy() database.ChunkingStrategy
}

// GetChunker returns the appropriate chunker for the given strategy.
func GetChunker(strategy database.ChunkingStrategy, embedder Embedder) (Chunker, error) {
	switch strategy {
	case database.ChunkingStrategyRecursive:
		return NewRecursiveChunker(512, 51), nil
	case database.ChunkingStrategyFixed:
		return NewFixedChunker(512, 51), nil
	case database.ChunkingStrategyParagraph:
		return &ParagraphChunker{}, nil
	case database.ChunkingStrategySemantic:
		return NewSemanticChunker(embedder, 0.70), nil
	default:
		return NewRecursiveChunker(512, 51), nil
	}
}

// ---- Recursive Chunker ----

var separators = []string{"\n\n", "\n", " ", ""}

type RecursiveChunker struct {
	chunkSize    int
	chunkOverlap int
}

func NewRecursiveChunker(size, overlap int) *RecursiveChunker {
	if size <= 0 {
		size = 512
	}
	if overlap < 0 {
		overlap = 51
	}
	return &RecursiveChunker{chunkSize: size, chunkOverlap: overlap}
}

func (c *RecursiveChunker) Strategy() database.ChunkingStrategy {
	return database.ChunkingStrategyRecursive
}

func (c *RecursiveChunker) Chunk(_ context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}
	splits := splitRecursive(text, separators, c.chunkSize)
	return mergeWithOverlap(splits, c.chunkSize, c.chunkOverlap), nil
}

func splitRecursive(text string, seps []string, maxSize int) []string {
	if len(text) <= maxSize || len(seps) == 0 {
		if text == "" {
			return nil
		}
		return []string{text}
	}

	sep := seps[0]
	nextSeps := seps[1:]

	if sep == "" {
		var parts []string
		for i := 0; i < len(text); i += maxSize {
			end := i + maxSize
			if end > len(text) {
				end = len(text)
			}
			parts = append(parts, text[i:end])
		}
		return parts
	}

	var parts []string
	for _, seg := range strings.Split(text, sep) {
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

func mergeWithOverlap(splits []string, maxSize, overlap int) []string {
	if len(splits) == 0 {
		return nil
	}

	var chunks []string
	var current strings.Builder

	for _, s := range splits {
		if current.Len() > 0 && current.Len()+1+len(s) > maxSize {
			chunk := strings.TrimSpace(current.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
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

	if chunk := strings.TrimSpace(current.String()); chunk != "" {
		chunks = append(chunks, chunk)
	}
	return chunks
}

// ---- Fixed Chunker ----

type FixedChunker struct {
	chunkSize    int
	chunkOverlap int
}

func NewFixedChunker(size, overlap int) *FixedChunker {
	if size <= 0 {
		size = 512
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	return &FixedChunker{chunkSize: size, chunkOverlap: overlap}
}

func (c *FixedChunker) Strategy() database.ChunkingStrategy { return database.ChunkingStrategyFixed }

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

type ParagraphChunker struct{}

func (c *ParagraphChunker) Strategy() database.ChunkingStrategy {
	return database.ChunkingStrategyParagraph
}

func (c *ParagraphChunker) Chunk(_ context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	var chunks []string
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.ReplaceAll(para, "\n", " ")
		para = strings.TrimSpace(para)
		if para != "" {
			chunks = append(chunks, para)
		}
	}
	return chunks, nil
}

// ---- Semantic Chunker ----

type SemanticChunker struct {
	embedder  Embedder
	threshold float64
}

func NewSemanticChunker(embedder Embedder, threshold float64) *SemanticChunker {
	if threshold <= 0 {
		threshold = 0.70
	}
	return &SemanticChunker{embedder: embedder, threshold: threshold}
}

func (c *SemanticChunker) Strategy() database.ChunkingStrategy {
	return database.ChunkingStrategySemantic
}

func (c *SemanticChunker) Chunk(ctx context.Context, text string) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("chunker: text must not be empty")
	}
	if c.embedder == nil {
		return nil, errors.New("chunker: embedder must not be nil for semantic chunking")
	}

	sentences := splitSentences(text)
	if len(sentences) <= 1 {
		return sentences, nil
	}

	embeddings, err := c.embedder.Embed(ctx, sentences)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != len(sentences) {
		return nil, errors.New("chunker: embedding count does not match sentence count")
	}

	var chunks []string
	start := 0
	for i := 1; i < len(sentences); i++ {
		sim := cosineSimilarity(embeddings[i-1], embeddings[i])
		if sim < c.threshold {
			chunks = append(chunks, strings.Join(sentences[start:i], " "))
			start = i
		}
	}
	if start < len(sentences) {
		chunks = append(chunks, strings.Join(sentences[start:], " "))
	}
	return chunks, nil
}

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
	if s := strings.TrimSpace(buf.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}

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
