package server

import (
	"context"
	"fmt"
	"net/http"
	"stockmind/internal/agent"
	"stockmind/internal/database"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port   int
	db     *database.Queries
	dbPool *pgxpool.Pool
	agent  *agent.AgentService
}

func NewServer(dbPool *pgxpool.Pool, agent *agent.AgentService, port string) *http.Server {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		portInt = 8080
	}
	NewServer := &Server{
		port:   portInt,
		db:     database.New(dbPool),
		dbPool: dbPool,
		agent:  agent,
	}

	// Ensure default user exists
	if err := NewServer.EnsureDefaultUser(); err != nil {
		fmt.Printf("Warning: Failed to ensure default user: %v\n", err)
	}

	// Ensure default agent flow is updated
	if err := NewServer.EnsureDefaultAgentFlow(); err != nil {
		fmt.Printf("Warning: Failed to ensure default agent flow: %v\n", err)
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
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

func (s *Server) EnsureDefaultAgentFlow() error {
	// Default flow ID from migration
	flowID := "01993ca8-a62e-79e3-995c-a46e25a4a2a2"
	// Update the config to use gpt-4o-mini instead of NEMOTRON_NANO_9B_V2
	query := `
		UPDATE agent_flows
		SET config = jsonb_set(config, '{agents,NormalChat,modelId}', '"gpt-4o-mini"')
		WHERE id = $1;
	`
	_, err := s.dbPool.Exec(context.Background(), query, flowID)
	return err
}
