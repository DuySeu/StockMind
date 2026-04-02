package rag

import (
	"context"
	"errors"
	"strings"
)

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
