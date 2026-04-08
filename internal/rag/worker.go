package rag

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"stockmind/internal/database"
)

// Job represents a background processing task for an uploaded document.
type Job struct {
	DocID    uuid.UUID
	Name     string
	FileType string
	Strategy Strategy
	TempFile string
}

// Worker orchestrates the background processing pipeline.
type Worker struct {
	db       *database.Queries
	store    Store
	embedder Embedder
	jobs     chan *Job
	wg       sync.WaitGroup
}

// NewWorker creates a new worker pool.
func NewWorker(db *database.Queries, store Store, embedder Embedder) *Worker {
	return &Worker{
		db:       db,
		store:    store,
		embedder: embedder,
		jobs:     make(chan *Job, 10), // Configured with buffered cap=10
	}
}

// Enqueue adds a job to the worker queue.
func (w *Worker) Enqueue(job *Job) {
	w.jobs <- job
}

// Start launches the background goroutine pool for processing documents.
// It will gracefully stop and finish pending jobs when ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	numWorkers := 2

	for i := 0; i < numWorkers; i++ {
		w.wg.Add(1)
		go func(workerID int) {
			defer w.wg.Done()
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d: shutting down gracefully", workerID)
					return
				case job := <-w.jobs:
					log.Printf("Worker %d: processing job %s", workerID, job.DocID)
					w.process(ctx, job)
				}
			}
		}(i)
	}
}

// Wait blocks until all active jobs finish, useful for graceful shutdown
// after ctx is cancelled.
func (w *Worker) Wait() {
	w.wg.Wait()
}

// process handles the full pipeline: Parse -> Chunk -> Embed -> Store.
func (w *Worker) process(ctx context.Context, job *Job) {
	defer os.Remove(job.TempFile) // Always cleanup temp file

	updateStatus := func(status string, count int, errMsg string) {
		msg := pgtype.Text{String: errMsg, Valid: errMsg != ""}
		err := w.db.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:         job.DocID,
			Status:     status,
			ChunkCount: int32(count),
			ErrorMsg:   msg,
		})
		if err != nil {
			log.Printf("Failed to update status to %s for doc %s: %v", status, job.DocID, err)
		}
	}

	updateStatus("processing", 0, "")

	parser, err := getParser(job.FileType)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("parser init: %v", err))
		return
	}

	file, err := os.Open(job.TempFile)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("failed to open file %s: %v", job.TempFile, err))
		return
	}
	defer file.Close()

	text, err := parser.Parse(file)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("parsing failed: %v", err))
		return
	}

	// Validate text
	validator := NewValidator()
	if err := validator.Validate(text); err != nil {
		updateStatus("failed", 0, fmt.Sprintf("validation failed: %v", err))
		return
	}

	chunker, err := getChunker(job.Strategy, w.embedder)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("chunker init: %v", err))
		return
	}

	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("chunking failed: %v", err))
		return
	}

	if len(chunks) == 0 {
		updateStatus("failed", 0, "No chunks generated from document")
		return
	}

	vectors, err := w.embedder.Embed(ctx, chunks)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("embedding failed: %v", err))
		return
	}

	if len(vectors) != len(chunks) {
		updateStatus("failed", 0, "Generated vector count does not match chunk count")
		return
	}

	err = w.store.Upsert(ctx, job.DocID.String(), chunks, vectors, job.Strategy)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("upsert to vector store failed: %v", err))
		return
	}

	updateStatus("ready", len(chunks), "")
}

func getParser(fileType string) (Parser, error) {
	switch fileType {
	case "pdf":
		return NewPDFParser(), nil
	case "docx":
		return NewDOCXParser(), nil
	case "md", "markdown":
		return NewMDParser(), nil
	case "txt", "text":
		return NewTXTParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func getChunker(strategy Strategy, embedder Embedder) (Chunker, error) {
	switch strategy {
	case StrategyRecursive:
		return NewRecursiveChunker(512, 51), nil
	case StrategyFixed:
		return NewFixedChunker(512, 51), nil
	case StrategyParagraph:
		return NewParagraphChunker(), nil
	case StrategySemantic:
		return NewSemanticChunker(embedder, 0.70), nil
	default:
		// Default to recursive if strategy is empty or invalid
		return NewRecursiveChunker(512, 51), nil
	}
}
