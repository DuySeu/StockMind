package service

import (
	"log"

	kb "stockmind/internal/knowledge"
	"stockmind/internal/service/tavily"
	"stockmind/internal/service/worker"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Services struct {
	Tavily         *tavily.Client
	DocWorker      *worker.DocumentWorker
	ResearchWorker *worker.ResearchWorker
}

func NewService(pipeline *kb.IngestPipeline, dbPool *pgxpool.Pool, store storage.ObjectStore) *Services {
	dw := worker.NewDocumentWorker(dbPool, pipeline, store)
	log.Println("Document worker pool initialized (elastic, max 2)")

	return &Services{
		Tavily:    tavily.NewClient(),
		DocWorker: dw,
	}
}

// InitResearchWorker initializes the research worker pool with the given process function.
// Must be called after the server is created since the process function depends on server methods.
func (s *Services) InitResearchWorker(processFunc func(*worker.ResearchJob)) {
	s.ResearchWorker = worker.NewResearchWorker(processFunc)
	log.Println("Research worker pool initialized (elastic, max 3)")
}

// Shutdown stops all worker pools and waits for in-flight jobs to finish.
func (s *Services) Shutdown() {
	s.DocWorker.Shutdown()
	if s.ResearchWorker != nil {
		s.ResearchWorker.Shutdown()
	}
}
