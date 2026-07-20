package knowledge

import (
	"context"
	"fmt"
	"io"
	"stockmind/internal/database"

	"github.com/google/uuid"
)

// IngestPipeline handles the full document processing: parse → chunk → embed → upsert.
type IngestPipeline struct {
	embedder  Embedder
	tokenizer *Tokenizer
	store     Store
}

// Process runs the full ingestion pipeline for a document.
func (p *IngestPipeline) Process(ctx context.Context, docID uuid.UUID, file io.Reader, fileType database.FileType, strategy database.ChunkingStrategy) (int, error) {
	parser, err := GetParser(fileType)
	if err != nil {
		return 0, fmt.Errorf("parser init: %w", err)
	}

	text, err := parser.Parse(file)
	if err != nil {
		return 0, fmt.Errorf("parsing failed: %w", err)
	}

	validator := NewValidator()
	if err := validator.Validate(text); err != nil {
		return 0, fmt.Errorf("validation failed: %w", err)
	}

	chunker, err := GetChunker(strategy, p.embedder)
	if err != nil {
		return 0, fmt.Errorf("chunker init: %w", err)
	}

	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		return 0, fmt.Errorf("chunking failed: %w", err)
	}
	if len(chunks) == 0 {
		return 0, fmt.Errorf("no chunks generated from document")
	}

	dense, err := p.embedder.Embed(ctx, chunks)
	if err != nil {
		return 0, fmt.Errorf("embedding failed: %w", err)
	}

	// Update BM25 document frequencies and generate sparse vectors
	p.tokenizer.AddDocuments(chunks)
	sparse := p.tokenizer.VectorizeBatch(chunks)

	if err := p.store.Upsert(ctx, docID, chunks, dense, sparse); err != nil {
		return 0, fmt.Errorf("upsert to vector store failed: %w", err)
	}

	return len(chunks), nil
}
