package worker

import (
	"time"

	"stockmind/internal/database"
)

// ResearchResult is the outcome of a single ticker research job.
type ResearchResult struct {
	Ticker string
	Report database.StockReport
	Err    error
}

// ResearchJob represents a single ticker research task.
type ResearchJob struct {
	Ticker        string
	ResearchModel string
	ProgressCh    chan<- any           // optional; worker sends progressEvent values here
	ResultCh      chan<- ResearchResult // worker sends exactly one result when done
}

// ResearchWorker wraps a generic Pool for market research processing.
type ResearchWorker struct {
	pool *Pool[*ResearchJob]
}

// NewResearchWorker creates a research worker pool.
// processFunc is provided by the caller (server layer) since it needs access to Tavily/LLM.
func NewResearchWorker(processFunc func(*ResearchJob)) *ResearchWorker {
	pool := NewPool[*ResearchJob](PoolConfig{
		MaxWorkers:  3,
		QueueSize:   20,
		IdleTimeout: 30 * time.Second,
	}, processFunc)

	return &ResearchWorker{pool: pool}
}

// Enqueue adds a research job to the pool.
func (rw *ResearchWorker) Enqueue(job *ResearchJob) error {
	return rw.pool.Enqueue(job)
}

// Shutdown stops the research worker pool.
func (rw *ResearchWorker) Shutdown() {
	rw.pool.Shutdown()
}
