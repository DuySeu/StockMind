package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	kb "stockmind/internal/knowledge_base"
	core "stockmind/internal/llm"
	"stockmind/internal/service"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mark3labs/mcp-go/mcp"
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
	toolMgr := core.NewToolManager()

	// Register retrieve_knowledge as internal tool
	toolMgr.Register(
		mcp.NewTool("retrieve_knowledge",
			mcp.WithDescription("Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Query related to financial knowledge or concepts")),
		),
		retrieveKnowledgeHandler(knowledgeBase.Retriever),
	)

	agent, err := core.NewLLMService(ctx, common.GetProviderName(), common.GetLLMModelName(), config.LLMConfig, dbPool, toolMgr)
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

func retrieveKnowledgeHandler(retriever kb.Retriever) core.ToolHandler {
	return func(ctx context.Context, args string) (string, error) {
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if strings.TrimSpace(params.Query) == "" {
			return "", fmt.Errorf("query is required")
		}

		results, err := retriever.Search(ctx, params.Query, kb.SearchHybrid, 5)
		if err != nil {
			return "", fmt.Errorf("knowledge base search failed: %w", err)
		}

		if len(results) == 0 {
			return "No relevant information found in knowledge base.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
		for i, res := range results {
			sb.WriteString(fmt.Sprintf("--- Source %d | Doc: %s, Chunk: %d ---\n", i+1, res.DocID, res.ChunkIndex))
			sb.WriteString(res.Text)
			sb.WriteString("\n\n")
		}

		return sb.String(), nil
	}
}
