package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"stockmind/internal/database"
	kb "stockmind/internal/knowledge"
	core "stockmind/internal/llm"
	"stockmind/internal/service"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ServerDeps holds all dependencies needed by the HTTP server.
type ServerDeps struct {
	DBPool      *pgxpool.Pool
	Agent       *core.LLMService
	KBStore     kb.Store
	ObjectStore storage.ObjectStore
	Services    *service.Services
}

type Server struct {
	queries        *database.Queries
	dbPool         *pgxpool.Pool
	agent          *core.LLMService
	knowledgeStore kb.Store
	objectStore    storage.ObjectStore
	services       *service.Services
}

func NewServer(deps ServerDeps, port string) *http.Server {
	srv := &Server{
		queries:        database.New(deps.DBPool),
		dbPool:         deps.DBPool,
		agent:          deps.Agent,
		knowledgeStore: deps.KBStore,
		objectStore:    deps.ObjectStore,
		services:       deps.Services,
	}

	// Initialize research worker pool now that the server (and its methods) exist.
	deps.Services.InitResearchWorker(srv.ProcessResearchJob)

	if err := srv.EnsureDefaultUser(); err != nil {
		log.Printf("Warning: Failed to ensure default user: %v\n", err)
	}

	return &http.Server{
		Addr:         ":" + port,
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Minute,
	}
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
