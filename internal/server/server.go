package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	kb "stockmind/internal/knowledge_base"
	core "stockmind/internal/llm"
	"stockmind/internal/service"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	queries        *database.Queries
	dbPool         *pgxpool.Pool
	agent          *core.LLMService
	knowledgeStore kb.Store
	objectStore    storage.ObjectStore
	service        *service.Service
}

func NewServer(ctx context.Context, config *common.Config, dbPool *pgxpool.Pool, knowledgeBase *kb.KnowledgeBase, objectStore storage.ObjectStore, service *service.Service, port string) *http.Server {
	agent, err := core.NewLLMService(ctx, common.GetProviderName(), common.GetLLMModelName(), config.LLMConfig, dbPool, knowledgeBase)
	if err != nil {
		log.Fatalf("failed to initialize LLM service: %v", err)
	}

	srv := &Server{
		queries:        database.New(dbPool),
		dbPool:         dbPool,
		agent:          agent,
		knowledgeStore: knowledgeBase.Store,
		objectStore:    objectStore,
		service:        service,
	}

	if err := srv.EnsureDefaultUser(); err != nil {
		log.Printf("Warning: Failed to ensure default user: %v\n", err)
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Minute,
	}

	return server
}

func (s *Server) EnsureDefaultUser() error {
	userID := "123e4567-e89b-12d3-a456-426614174000"
	query := `
		INSERT INTO users (id, name, email, provider)
		VALUES ($1, 'Default User', 'default@example.com', 'system')
		ON CONFLICT (id) DO NOTHING;
	`
	_, err := s.dbPool.Exec(context.Background(), query, userID)
	return err
}
