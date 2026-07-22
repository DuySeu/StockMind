package worker

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrQueueFull = errors.New("worker: job queue is full")

// Pool is a generic elastic worker pool that processes jobs of any type.
type Pool[T any] struct {
	process       func(T)
	jobs          chan T
	mu            sync.Mutex
	activeWorkers int
	maxWorkers    int
	idleTimeout   time.Duration
	wg            sync.WaitGroup
}

// PoolConfig holds configuration for a worker pool.
type PoolConfig struct {
	MaxWorkers  int
	QueueSize   int
	IdleTimeout time.Duration
}

// NewPool creates a new generic elastic worker pool.
func NewPool[T any](cfg PoolConfig, process func(T)) *Pool[T] {
	return &Pool[T]{
		process:     process,
		jobs:        make(chan T, cfg.QueueSize),
		maxWorkers:  cfg.MaxWorkers,
		idleTimeout: cfg.IdleTimeout,
	}
}

// Enqueue adds a job to the pool and spawns a worker if needed.
func (p *Pool[T]) Enqueue(job T) error {
	select {
	case p.jobs <- job:
		p.trySpawn()
		return nil
	default:
		return ErrQueueFull
	}
}

// Shutdown closes the job channel and waits for in-flight jobs to complete.
func (p *Pool[T]) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *Pool[T]) trySpawn() {
	p.mu.Lock()
	if p.activeWorkers >= p.maxWorkers {
		p.mu.Unlock()
		return
	}
	p.activeWorkers++
	p.mu.Unlock()

	p.wg.Add(1)
	go p.run()
}

func (p *Pool[T]) run() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("worker: recovered from panic", "panic", r)
		}
		p.mu.Lock()
		p.activeWorkers--
		p.mu.Unlock()
		p.wg.Done()
	}()

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.process(job)
		case <-time.After(p.idleTimeout):
			return
		}
	}
}
