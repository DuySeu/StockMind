package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/tool"
	"stockmind/internal/mcp"
	"stockmind/internal/rag"
	"stockmind/internal/server"
	"stockmind/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

func main() {
	// Load .env file first
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Failed to load .env file: %v\n", err)
	}

	app := cli.Command{
		Name:    "stock_mind",
		Usage:   "StockMind is an AI-powered assistant designed to simplify access to financial information and insights about the Vietnamese stock market.",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:  "server-new",
				Usage: "Run the LLM Core API server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Port to run the server on",
					},
					&cli.StringFlag{
						Name:    "mcp-protocol",
						Aliases: []string{"p"},
						Usage:   "MCP protocol (stdio, http)",
						Value:   "stdio",
					},
				},
				Action: runServerCmd,
			},
			{
				Name:  "server",
				Usage: "Run the StockMind application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Port to run the server on",
					},
					&cli.StringFlag{
						Name:    "mcp-protocol",
						Aliases: []string{"p"},
						Usage:   "MCP protocol (stdio, http)",
						Value:   "stdio",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					port := cmd.String("port")
					mcpProtocol := cmd.String("mcp-protocol")

					runContext, shutdown, err := runServer(ctx, port, mcpProtocol)
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
			{
				Name:  "mcp",
				Usage: "Run the MCP example",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "protocol",
						Aliases: []string{"p"},
						Usage:   "Protocol to use (stdio, http). http means Streamable HTTP protocol",
						Value:   "stdio",
						Sources: cli.ValueSourceChain{
							Chain: []cli.ValueSource{
								cli.EnvVar("MCP_PROTOCOL"),
							},
						},
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					protocol := cmd.String("protocol")
					err := runMCP(ctx, protocol)
					return err
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func runMCP(ctx context.Context, protocol string) error {
	log.Printf("Running MCP server with protocol: %s", protocol)
	_, err := mcp.Start(ctx, protocol, nil, nil)
	return err
}

func runServer(ctx context.Context, port string, mcpProtocol string) (context.Context, func(), error) {
	log.Printf("Running server on port: %s", port)

	// Will initialize MCP HTTP server after Qdrant/Embedder if needed

	dbUrl := "postgres://" + os.Getenv("DB_USERNAME") + ":" + url.QueryEscape(os.Getenv("DB_PASSWORD")) + "@" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + "/" + os.Getenv("DB_DATABASE") + "?sslmode=disable"

	// Create a database connection pool
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = 10

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database pool: %v", err)
	}

	// Test the database connection
	err = dbPool.Ping(ctx)
	if err != nil {
		log.Printf("Failed to ping database: %v", err)
		return nil, nil, err
	}

	// Run Migration
	err = database.MigrateDB(dbPool)
	if err != nil {
		log.Println("Failed to migrate database", "error", err)
		return nil, nil, err
	}
	log.Println("Database connection established")

	// Initialize Qdrant Client
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost"
	}
	qdrantPort := os.Getenv("QDRANT_PORT")
	if qdrantPort == "" {
		qdrantPort = "6334"
	}

	qdrantConn, err := rag.InitQdrant(ctx, qdrantHost, qdrantPort)
	if err != nil {
		log.Fatalf("Failed to initialize Qdrant: %v", err)
	}
	qdrantStore := rag.NewQdrantStore(qdrantConn)

	// Initialize OpenRouter Embedder
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	embedder, err := rag.NewOpenRouterEmbedder(openrouterKey, 20)
	if err != nil {
		log.Printf("Warning: Failed to initialize embedder: %v", err)
	}

	var mcpShutdown func()
	if mcpProtocol == "http" {
		// Create MCP service and HTTP server
		log.Printf("Initializing MCP server with HTTP protocol on 0.0.0.0:8081")
		shutdown, err := mcp.Start(ctx, mcpProtocol, qdrantStore, embedder)
		if err != nil {
			log.Printf("Failed to start MCP: %v", err)
			return nil, nil, err
		}
		mcpShutdown = shutdown
	}

	// Initialize Worker
	dbQueries := database.New(dbPool)
	worker := rag.NewWorker(dbQueries, qdrantStore, embedder)

	// Start the worker pool
	workerCtx, workerCancel := context.WithCancel(ctx)
	worker.Start(workerCtx)

	// Create LLM Config
	// llmConfig := agent.LoadLLMConfig()

	// Create an agent service
	// _, err = agent.NewService(ctx, dbPool, llmConfig)
	// if err != nil {
	// 	log.Println("Failed to create agent service", "error", err)
	// 	workerCancel() // Stop worker pool on failure to start server
	// 	return nil, nil, err
	// }

	// Create Document Service
	documentService := service.NewDocumentService(dbQueries, worker, qdrantStore)

	// Create a server for the application
	server := server.NewServer(dbPool, nil, documentService, port)
	runContext, cancel := context.WithCancel(ctx)

	// Create a done channel to signal when the shutdown is complete
	stopCh := make(chan struct{})

	go func() {
		defer close(stopCh)

		log.Printf("Server starting on port: %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("unexpected server closure: %v", err)
			cancel()
		}
	}()

	return runContext, func() {
		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// Shutdown MCP server first if it was started
		if mcpShutdown != nil {
			log.Printf("Shutting down MCP server...")
			mcpShutdown()
		}

		// Shutdown HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("stopping server: %v", err)
		}

		// Wait for worker pool to finish in-flight jobs gracefully
		log.Printf("Shutting down async worker pool...")
		workerCancel()
		worker.Wait()

		<-stopCh
	}, nil
}

func runServerCmd(ctx context.Context, cmd *cli.Command) error {
	port := cmd.String("port")

	dbUrl := "postgres://" + os.Getenv("DB_USERNAME") + ":" + url.QueryEscape(os.Getenv("DB_PASSWORD")) + "@" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + "/" + os.Getenv("DB_DATABASE") + "?sslmode=disable"

	// Create a database connection pool
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = 10
	// Create dependencies
	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	log.Printf("Database: OK (pool ping succeeded)")

	// Run Migration
	if err := database.MigrateDB(dbPool); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	log.Printf("Database: Migration complete")

	cfg := core.LoadLLMConfig()

	toolMgr := tool.NewToolManager()

	if mcpArg := strings.TrimSpace(cmd.String("mcp")); mcpArg != "" {
		appPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable for MCP: %w", err)
		}
		mcpArgs := strings.Fields(mcpArg)
		mcpClient, err := tool.NewMCPClient(appPath, mcpArgs...)
		if err != nil {
			return fmt.Errorf("MCP server failed to start: %w", err)
		}
		defer mcpClient.Close()

		mcpTools, err := mcpClient.GetTools(ctx)
		if err != nil {
			return fmt.Errorf("MCP server tools/list failed: %w", err)
		}
		log.Printf("MCP server: OK (handshake + tools/list; %d tool(s))", len(mcpTools))

		for _, t := range mcpTools {
			toolName := t.Name
			toolMgr.Register(t, func(ctx context.Context, args string) (string, error) {
				return mcpClient.CallTool(ctx, toolName, args)
			})
		}
	} else {
		log.Printf("MCP server: not configured (pass --mcp \"mcp-search\" to attach a stdio MCP subprocess)")
	}
	historyRepo := database.New(dbPool)
	agent, err := core.NewLLMService(ctx, core.GetProviderName(), core.GetModelName(), cfg, historyRepo, toolMgr)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	srv := server.NewServer(dbPool, agent, nil, port)

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}
	log.Printf("Server ready; listening on %s", ln.Addr())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}
