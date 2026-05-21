package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	kb "stockmind/internal/knowledge_base"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/tools"
	"stockmind/internal/mcp"
	"stockmind/internal/server"
	"stockmind/internal/service"
	"stockmind/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Failed to load .env file: %v\n", err)
	}

	app := cli.Command{
		Name:    "stock_mind",
		Usage:   "StockMind is an AI-powered assistant designed to simplify access to financial information and insights about the Vietnamese stock market.",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Run the StockMind application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Port to run the server on",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					port := cmd.String("port")

					runContext, shutdown, err := runServer(ctx, port)
					if err != nil {
						log.Printf("Failed to run server: %v", err)
						return err
					}

					signalCtx, cancel := signal.NotifyContext(runContext, syscall.SIGINT, syscall.SIGTERM)
					defer cancel()

					<-signalCtx.Done()
					shutdown()
					return nil
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func runServer(ctx context.Context, port string) (context.Context, func(), error) {
	log.Printf("Running server on port: %s", port)

	config := common.LoadConfig()

	poolConfig, err := pgxpool.ParseConfig(config.GetDBURL())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = 10

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database pool: %v", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("database ping failed: %w", err)
	}

	if err := database.MigrateDB(dbPool); err != nil {
		return nil, nil, fmt.Errorf("database migration failed: %w", err)
	}
	log.Println("Database connection established")

	// Initialize MinIO object store
	objectStore, err := storage.NewMinIOStore(ctx, config.MinIO)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init object store: %w", err)
	}
	log.Println("MinIO object store ready")

	// Initialize Knowledge Base (Qdrant + Embedder + BM25)
	knowledgeBase, err := kb.New(ctx, &config, dbPool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init knowledge base: %w", err)
	}

	// Initialize MCP client manager and dynamically bridge MCP tools
	var mcpManager *mcp.Manager
	var bridgedMCPTools []*tools.Tool

	var mcpConfigs []mcp.ServerConfig
	if _, err := exec.LookPath("uvx"); err == nil {
		mcpConfigs = append(mcpConfigs, mcp.ServerConfig{
			Name:    "aws-docs",
			Command: "uvx",
			Args: []string{
				"awslabs.aws-documentation-mcp-server@latest",
			},
			Env: map[string]string{
				"FASTMCP_LOG_LEVEL":           "ERROR",
				"AWS_DOCUMENTATION_PARTITION": "aws",
			},
		})
	} else {
		log.Println("Warning: uvx not found — external MCP clients like AWS documentation disabled")
	}

	if len(mcpConfigs) > 0 {
		mcpManager = mcp.NewManager(mcpConfigs)
		var bridgeErr error
		bridgedMCPTools, bridgeErr = tools.BridgeMCPTools(ctx, mcpManager)
		if bridgeErr != nil {
			log.Printf("Warning: failed to bridge MCP tools: %v", bridgeErr)
		} else {
			log.Printf("Successfully bridged %d dynamic MCP tools", len(bridgedMCPTools))
		}
	}

	// Initialize Services
	services := service.NewService(knowledgeBase.Pipeline, dbPool, objectStore)

	// Initialize tools and LLM service
	toolDefs := tools.RegisterTools(knowledgeBase.Retriever, services)
	if len(bridgedMCPTools) > 0 {
		toolDefs = append(toolDefs, bridgedMCPTools...)
	}
	toolMgr := tools.NewManager(toolDefs)

	agent, err := core.NewLLMService(ctx, common.GetProviderName(), common.GetLLMModelName(), config.LLMConfig, dbPool, toolMgr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init LLM service: %w", err)
	}

	// Create HTTP server
	srv := server.NewServer(server.ServerDeps{
		DBPool:      dbPool,
		Agent:       agent,
		KBStore:     knowledgeBase.Store,
		ObjectStore: objectStore,
		Services:    services,
	}, port)

	runContext, cancel := context.WithCancel(ctx)
	stopCh := make(chan struct{})

	go func() {
		defer close(stopCh)
		log.Printf("Server starting on port: %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("unexpected server closure: %v", err)
			cancel()
		}
	}()

	return runContext, func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// 1. Stop accepting new HTTP requests, drain in-flight
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("stopping server: %v", err)
		}

		// 2. Stop worker pool (drain queued jobs)
		log.Printf("Shutting down async worker pool...")
		services.Shutdown()

		// 3. Stop all external MCP client sessions
		if mcpManager != nil {
			log.Printf("Shutting down all active external MCP client sessions...")
			mcpManager.CloseAll()
		}

		<-stopCh
	}, nil
}
