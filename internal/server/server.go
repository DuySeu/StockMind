package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"stockmind/internal/agent"
	"stockmind/internal/database"
	"stockmind/internal/service/tavily"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port          int
	db            *database.Queries
	dbPool        *pgxpool.Pool
	agent         *agent.AgentService
	tavily        *tavily.Client
	streamManager *StreamManager
}

func NewServer(dbPool *pgxpool.Pool, agent *agent.AgentService, streamManager *StreamManager, port string) *http.Server {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		portInt = 8080
	}
	srv := &Server{
		port:          portInt,
		db:            database.New(dbPool),
		dbPool:        dbPool,
		agent:         agent,
		tavily:        tavily.NewClient(),
		streamManager: streamManager,
	}

	// Ensure default user exists
	if err := srv.EnsureDefaultUser(); err != nil {
		log.Printf("Warning: Failed to ensure default user: %v\n", err)
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.port),
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
