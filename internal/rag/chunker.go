package rag

import "context"

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
