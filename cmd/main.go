package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"stockmind/internal/agents"
	"stockmind/internal/common"
	"stockmind/internal/database"
	kb "stockmind/internal/knowledge"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"
	"stockmind/internal/mcp"
	"stockmind/internal/orchestration"
	"stockmind/internal/server"
	"stockmind/internal/service"
	"stockmind/internal/storage"
	"stockmind/internal/tools"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Failed to load .env file: %v\n", err)
	}

	common.InitLogging()

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
						slog.Error("failed to run server", "error", err)
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
	slog.Info("running server", "port", port)

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
	slog.Info("database connection established")

	// Initialize MinIO object store
	objectStore, err := storage.NewMinIOStore(ctx, config.MinIO)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init object store: %w", err)
	}
	slog.Info("minio object store ready")

	// Initialize Knowledge Base (Qdrant + Embedder + BM25)
	kbBase, err := kb.New(ctx, &config, dbPool)
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
		slog.Warn("uvx not found — external MCP clients like AWS documentation disabled")
	}

	if len(mcpConfigs) > 0 {
		mcpManager = mcp.NewManager(mcpConfigs)

		bridgeCtx, bridgeCancel := context.WithTimeout(ctx, 30*time.Second)
		defer bridgeCancel()

		bridgedMCPTools, err := tools.BridgeMCPTools(bridgeCtx, mcpManager)
		if err != nil {
			slog.Warn("failed to bridge MCP tools", "error", err)
		} else {
			slog.Info("bridged dynamic MCP tools", "count", len(bridgedMCPTools))
		}
	}

	// Initialize Services
	services := service.NewService(kbBase.Pipeline, dbPool, objectStore)

	// Initialize tools and LLM service
	toolDefs := tools.RegisterTools(kbBase.Retriever, services)
	if len(bridgedMCPTools) > 0 {
		toolDefs = append(toolDefs, bridgedMCPTools...)
	}
	toolMgr := tools.NewManager(toolDefs)

	agent, err := core.NewLLMService(ctx, config.LLM, toolMgr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init LLM service: %w", err)
	}

	// Multi-agent pipeline: specialist agent roster + planner + sequential executor.
	agentDeps := agents.Deps{LLM: agent, Prompts: prompts.NewPromptLoader()}
	agentRegistry := agents.NewRegistry(agentDeps)
	orchestrator := orchestration.New(agents.NewPlanner(agentDeps), agentRegistry)
	slog.Info("agent pipeline ready", "agents", agentRegistry.Names())

	// Create HTTP server
	srv := server.NewServer(server.ServerDeps{
		DBPool:       dbPool,
		Agent:        agent,
		KBStore:      kbBase.Store,
		ObjectStore:  objectStore,
		Services:     services,
		Orchestrator: orchestrator,
	}, port)

	runContext, cancel := context.WithCancel(ctx)
	stopCh := make(chan struct{})

	go func() {
		defer close(stopCh)
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("unexpected server closure", "error", err)
			cancel()
		}
	}()

	return runContext, func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// 1. Stop accepting new HTTP requests, drain in-flight
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("stopping server", "error", err)
		}

		// 2. Stop worker pool (drain queued jobs)
		slog.Info("shutting down async worker pool")
		services.Shutdown()

		// 3. Stop all external MCP client sessions
		if mcpManager != nil {
			slog.Info("shutting down external MCP client sessions")
			mcpManager.CloseAll()
		}

		<-stopCh
	}, nil
}
