package server

import (
	"context"
	"log"
	"net/http"
	"stockmind/internal/common"
	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/qdrant"
	"stockmind/internal/service"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	queries     *database.Queries
	dbPool      *pgxpool.Pool
	agent       *core.LLMService
	vectorStore *qdrant.QdrantStore
	service     *service.Service
}

func NewServer(ctx context.Context, config *common.Config, dbPool *pgxpool.Pool, vectorStore *qdrant.QdrantStore, service *service.Service, port string) *http.Server {
	// Create LLM Config
	toolMgr := core.NewToolManager()

	// Create an agent service
	agent, err := core.NewLLMService(ctx, common.GetProviderName(), common.GetModelName(), config.LLMConfig, dbPool, toolMgr)
	if err != nil {
		log.Fatalf("failed to initialize LLM service: %v", err)
	}

	srv := &Server{
		queries:     database.New(dbPool),
		dbPool:      dbPool,
		agent:       agent,
		vectorStore: vectorStore,
		service:     service,
	}

	// Ensure default user exists
	if err := srv.EnsureDefaultUser(); err != nil {
		log.Printf("Warning: Failed to ensure default user: %v\n", err)
	}

	// Declare Server config
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Minute, // increased for SSE streaming (research can take 2+ min)
	}

	return server
}

func (s *Server) EnsureDefaultUser() error {
	// Default user ID from routes.go
	userID := "123e4567-e89b-12d3-a456-426614174000"
	query := `
		INSERT INTO users (id, name, email, provider)
		VALUES ($1, 'Default User', 'default@example.com', 'system')
		ON CONFLICT (id) DO NOTHING;
	`
	_, err := s.dbPool.Exec(context.Background(), query, userID)
	return err
}
