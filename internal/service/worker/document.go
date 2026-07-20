package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"stockmind/internal/database"
	kb "stockmind/internal/knowledge"
	"stockmind/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentJob represents a background processing task for an uploaded document.
type DocumentJob struct {
	DocID     uuid.UUID
	Name      string
	FileType  database.FileType
	Strategy  database.ChunkingStrategy
	ObjectKey string
}

// DocumentWorker wraps a generic Pool for document processing.
type DocumentWorker struct {
	pool *Pool[*DocumentJob]
}

// NewDocumentWorker creates a document processing worker pool.
func NewDocumentWorker(dbPool *pgxpool.Pool, pipeline *kb.IngestPipeline, store storage.ObjectStore) *DocumentWorker {
	db := database.New(dbPool)
	processor := &docProcessor{db: db, pipeline: pipeline, store: store}

	pool := NewPool[*DocumentJob](PoolConfig{
		MaxWorkers:  2,
		QueueSize:   20,
		IdleTimeout: 10 * time.Second,
	}, processor.process)

	return &DocumentWorker{pool: pool}
}

// Enqueue adds a document job to the pool.
func (dw *DocumentWorker) Enqueue(job *DocumentJob) error {
	return dw.pool.Enqueue(job)
}

// Shutdown stops the document worker pool.
func (dw *DocumentWorker) Shutdown() {
	dw.pool.Shutdown()
}

type docProcessor struct {
	db       *database.Queries
	pipeline *kb.IngestPipeline
	store    storage.ObjectStore
}

func (p *docProcessor) process(job *DocumentJob) {
	ctx := context.Background()

	updateStatus := func(status database.DocumentStatus, count int, errMsg string) {
		msg := pgtype.Text{String: errMsg, Valid: errMsg != ""}
		if err := p.db.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:         job.DocID,
			Status:     status,
			ChunkCount: int32(count),
			ErrorMsg:   msg,
		}); err != nil {
			log.Printf("worker: failed to update status to %s for doc %s: %v", status, job.DocID, err)
		}
	}

	updateStatus("processing", 0, "")

	reader, err := p.store.Get(ctx, job.ObjectKey)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("download from storage: %v", err))
		return
	}
	defer reader.Close()

	chunkCount, err := p.pipeline.Process(ctx, job.DocID, reader, job.FileType, job.Strategy)
	if err != nil {
		updateStatus("failed", 0, err.Error())
		return
	}

	updateStatus("ready", chunkCount, "")
}
