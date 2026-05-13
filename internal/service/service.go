package service

import (
	"context"
	"fmt"
	"log"

	"stockmind/internal/common"
	llm "stockmind/internal/llm"
	"stockmind/internal/llm/rag"
	"stockmind/internal/qdrant"
	"stockmind/internal/service/tavily"
	"stockmind/internal/service/worker"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Service struct {
	Tavily       *tavily.Client
	Worker       *worker.Worker
	workerCancel context.CancelFunc
}

func NewService(ctx context.Context, cfg *common.Config, store *qdrant.QdrantStore, dbPool *pgxpool.Pool) (*Service, error) {
	orClient, err := llm.NewOpenRouterClient(cfg.LLMConfig.OpenRouter)
	if err != nil {
		return nil, fmt.Errorf("worker: failed to init openrouter client: %w", err)
	}

	embedder := rag.NewOpenRouterEmbedder(orClient, 20)

	w := worker.NewWorker(dbPool, store, embedder)

	workerCtx, workerCancel := context.WithCancel(ctx)
	w.Start(workerCtx)
	log.Println("Worker started")

	return &Service{
		Tavily:       tavily.NewClient(),
		Worker:       w,
		workerCancel: workerCancel,
	}, nil
}

// Shutdown stops the worker pool and waits for in-flight jobs to finish.
func (s *Service) Shutdown() {
	s.workerCancel()
	s.Worker.Wait()
}
