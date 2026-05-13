package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"stockmind/internal/database"
	"stockmind/internal/llm/rag"
	"stockmind/internal/qdrant"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Job represents a background processing task for an uploaded document.
type Job struct {
	DocID    uuid.UUID
	Name     string
	FileType string
	Strategy rag.Strategy
	TempFile string
}

// Worker orchestrates the background document processing pipeline.
type Worker struct {
	db       *database.Queries
	store    qdrant.Store
	embedder rag.Embedder
	jobs     chan *Job
	wg       sync.WaitGroup
}

// NewWorker creates a new worker pool.
func NewWorker(dbPool *pgxpool.Pool, store qdrant.Store, embedder rag.Embedder) *Worker {
	return &Worker{
		db:       database.New(dbPool),
		store:    store,
		embedder: embedder,
		jobs:     make(chan *Job, 10),
	}
}

// Enqueue adds a job to the worker queue.
func (w *Worker) Enqueue(job *Job) {
	w.jobs <- job
}

// Start launches the background goroutine pool.
func (w *Worker) Start(ctx context.Context) {
	numWorkers := 2
	for i := 0; i < numWorkers; i++ {
		w.wg.Add(1)
		go func(id int) {
			defer w.wg.Done()
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d: shutting down gracefully", id)
					return
				case job := <-w.jobs:
					log.Printf("Worker %d: processing job %s", id, job.DocID)
					w.process(ctx, job)
				}
			}
		}(i)
	}
}

// Wait blocks until all active jobs finish.
func (w *Worker) Wait() {
	w.wg.Wait()
}

func (w *Worker) process(ctx context.Context, job *Job) {
	defer os.Remove(job.TempFile)

	updateStatus := func(status string, count int, errMsg string) {
		msg := pgtype.Text{String: errMsg, Valid: errMsg != ""}
		if err := w.db.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:         job.DocID,
			Status:     status,
			ChunkCount: int32(count),
			ErrorMsg:   msg,
		}); err != nil {
			log.Printf("Failed to update status to %s for doc %s: %v", status, job.DocID, err)
		}
	}

	updateStatus("processing", 0, "")

	parser, err := rag.GetParser(job.FileType)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("parser init: %v", err))
		return
	}

	file, err := os.Open(job.TempFile)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("failed to open file: %v", err))
		return
	}
	defer file.Close()

	text, err := parser.Parse(file)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("parsing failed: %v", err))
		return
	}

	validator := rag.NewValidator()
	if err := validator.Validate(text); err != nil {
		updateStatus("failed", 0, fmt.Sprintf("validation failed: %v", err))
		return
	}

	chunker, err := rag.GetChunker(job.Strategy, w.embedder)
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

	err = w.store.Upsert(ctx, job.DocID.String(), chunks, vectors, string(job.Strategy))
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("upsert to vector store failed: %v", err))
		return
	}

	updateStatus("ready", len(chunks), "")
}
