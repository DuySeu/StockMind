package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/mcp"
	"stockmind/internal/qdrant"
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
	_, err := mcp.Start(ctx, protocol)
	return err
}

func runServer(ctx context.Context, port string, mcpProtocol string) (context.Context, func(), error) {
	log.Printf("Running server on port: %s", port)

	// Load Config
	config := common.LoadConfig()

	// Create a database connection pool
	poolConfig, err := pgxpool.ParseConfig(config.GetDBURL())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = 10

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database pool: %v", err)
	}

	// Test the database connection
	if err := dbPool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("database ping failed: %w", err)
	}

	// Run Migration
	if err := database.MigrateDB(dbPool); err != nil {
		return nil, nil, fmt.Errorf("database migration failed: %w", err)
	}
	log.Println("Database connection established")

	// Initialize Qdrant Client
	qdrantConn, err := qdrant.InitQdrant(ctx, config.Qdrant.Host, config.Qdrant.Port)
	if err != nil {
		log.Fatalf("Failed to initialize Qdrant: %v", err)
	}
	qdrantStore := qdrant.NewQdrantStore(qdrantConn, "nvidia/llama-nemotron-embed-vl-1b-v2:free")

	var mcpShutdown func()
	if mcpProtocol == "http" {
		// Create MCP service and HTTP server
		log.Printf("Initializing MCP server with HTTP protocol on 0.0.0.0:8081")
		shutdown, err := mcp.Start(ctx, mcpProtocol)
		if err != nil {
			log.Printf("Failed to start MCP: %v", err)
			return nil, nil, err
		}
		mcpShutdown = shutdown
	}

	// Initialize Worker
	services, err := service.NewService(ctx, &config, qdrantStore, dbPool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init service: %w", err)
	}

	// Create a server for the application
	server := server.NewServer(ctx, &config, dbPool, qdrantStore, services, port)
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
		services.Shutdown()

		<-stopCh
	}, nil
}
