package service

import (
	"log"

	kb "stockmind/internal/knowledge_base"
	"stockmind/internal/service/tavily"
	"stockmind/internal/service/worker"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Service struct {
	Tavily *tavily.Client
	Worker *worker.Worker
}

func NewService(pipeline *kb.IngestPipeline, dbPool *pgxpool.Pool, store storage.ObjectStore) *Service {
	w := worker.NewWorker(dbPool, pipeline, store)
	log.Println("Worker pool initialized (elastic, max 2)")

	return &Service{
		Tavily: tavily.NewClient(),
		Worker: w,
	}
}

// Shutdown stops the worker pool and waits for in-flight jobs to finish.
func (s *Service) Shutdown() {
	s.Worker.Shutdown()
}
