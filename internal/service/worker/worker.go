package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"stockmind/internal/database"
	kb "stockmind/internal/knowledge_base"
	"stockmind/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxWorkers        = 2
	workerIdleTimeout = 10 * time.Second
	queueSize         = 20
)

var ErrQueueFull = errors.New("worker: job queue is full")

// Job represents a background processing task for an uploaded document.
type Job struct {
	DocID     uuid.UUID
	Name      string
	FileType  database.FileType
	Strategy  database.ChunkingStrategy
	ObjectKey string
}

// Worker orchestrates the elastic background document processing pool.
type Worker struct {
	db            *database.Queries
	pipeline      *kb.IngestPipeline
	store         storage.ObjectStore
	jobs          chan *Job
	mu            sync.Mutex
	activeWorkers int
	wg            sync.WaitGroup
}

// NewWorker creates a new elastic worker pool.
func NewWorker(dbPool *pgxpool.Pool, pipeline *kb.IngestPipeline, store storage.ObjectStore) *Worker {
	return &Worker{
		db:       database.New(dbPool),
		pipeline: pipeline,
		store:    store,
		jobs:     make(chan *Job, queueSize),
	}
}

// Enqueue adds a job to the worker queue and spawns a worker if needed.
func (w *Worker) Enqueue(job *Job) error {
	select {
	case w.jobs <- job:
		w.trySpawn()
		return nil
	default:
		return ErrQueueFull
	}
}

// Shutdown closes the job channel and waits for in-flight jobs to complete.
func (w *Worker) Shutdown() {
	close(w.jobs)
	w.wg.Wait()
}

func (w *Worker) trySpawn() {
	w.mu.Lock()
	if w.activeWorkers >= maxWorkers {
		w.mu.Unlock()
		return
	}
	w.activeWorkers++
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()
}

func (w *Worker) run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("worker: recovered from panic: %v", r)
		}
		w.mu.Lock()
		w.activeWorkers--
		w.mu.Unlock()
		w.wg.Done()
	}()

	for {
		select {
		case job, ok := <-w.jobs:
			if !ok {
				return
			}
			w.process(job)
		case <-time.After(workerIdleTimeout):
			return
		}
	}
}

func (w *Worker) process(job *Job) {
	ctx := context.Background()

	updateStatus := func(status database.DocumentStatus, count int, errMsg string) {
		msg := pgtype.Text{String: errMsg, Valid: errMsg != ""}
		if err := w.db.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:         job.DocID,
			Status:     status,
			ChunkCount: int32(count),
			ErrorMsg:   msg,
		}); err != nil {
			log.Printf("worker: failed to update status to %s for doc %s: %v", status, job.DocID, err)
		}
	}

	updateStatus("processing", 0, "")

	reader, err := w.store.Get(ctx, job.ObjectKey)
	if err != nil {
		updateStatus("failed", 0, fmt.Sprintf("download from storage: %v", err))
		return
	}
	defer reader.Close()

	chunkCount, err := w.pipeline.Process(ctx, job.DocID, reader, job.FileType, job.Strategy)
	if err != nil {
		updateStatus("failed", 0, err.Error())
		return
	}

	updateStatus("ready", chunkCount, "")
}
