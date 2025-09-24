package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockmind/internal/agent"
	"stockmind/internal/database"
	"stockmind/internal/mcp"
	"stockmind/internal/server"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urfave/cli/v3"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
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
					return runServer(ctx, port)
				},
			},
			{
				Name:  "mcp",
				Usage: "Run the MCP server",
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
					return runMCP(ctx, protocol)
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func runServer(ctx context.Context, port string) error {
	log.Printf("Running server on port: %s", port)

	dbUrl := "postgres://" + os.Getenv("DB_USERNAME") + ":" + url.QueryEscape(os.Getenv("DB_PASSWORD")) + "@" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + "/" + os.Getenv("DB_DATABASE") + "?sslmode=disable"

	// Create a database connection pool
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = 10

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %v", err)
	}

	// Test the database connection
	err = dbPool.Ping(ctx)
	if err != nil {
		log.Printf("Failed to ping database: %v", err)
		return err
	}

	// Run Migration
	err = database.MigrateDB(dbPool)
	if err != nil {
		log.Println("Failed to migrate database", "error", err)
		return err
	}
	log.Println("Database connection established")

	// Create an agent service
	agent, err := agent.NewService(ctx, dbPool, database.ModelProviderOpenAI)
	if err != nil {
		log.Println("Failed to create agent service", "error", err)
		return err
	}

	// Create a server for the application
	server := server.NewServer(dbPool, agent)

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %s", err)
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")

	return nil
}

func runMCP(ctx context.Context, protocol string) error {
	log.Printf("Running MCP server with protocol: %s", protocol)
	return mcp.Start(ctx, protocol)
}
