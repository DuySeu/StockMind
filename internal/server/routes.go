package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	core "stockmind/internal/llm"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

type Message struct {
	Content string `json:"content"`
}

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", s.HelloWorldHandler)
	r.Get("/health", s.healthHandler)

	r.Route("/v1", func(r chi.Router) {
		// Websocket
		r.Get("/ws", s.websocketHandler)
		r.Post("/llm", s.llmHandler)

		// Users
		r.Route("/users", func(r chi.Router) {
			r.Post("/", s.CreateUserHandler)
			r.Get("/", s.GetUsersHandler)
			r.Get("/{id}", s.GetUserByIDHandler)
			r.Put("/{id}", s.UpdateUserHandler)
			r.Delete("/{id}", s.DeleteUserHandler)
		})

		// Threads
		// r.Route("/threads", func(r chi.Router) {
		// 	r.Post("/", s.CreateThreadHandler)
		// 	r.Get("/", s.GetThreadsHandler)
		// 	r.Get("/{id}", s.GetThreadByIDHandler)
		// 	r.Put("/{id}", s.UpdateThreadHandler)
		// 	r.Delete("/{id}", s.DeleteThreadHandler)
		// })

		// Messages
		// r.Route("/messages", func(r chi.Router) {
		// 	r.Post("/", s.CreateMessageHandler)
		// 	r.Get("/", s.GetMessagesHandler)
		// 	r.Get("/{id}", s.GetMessageByIDHandler)
		// 	r.Put("/{id}", s.UpdateMessageHandler)
		// 	r.Delete("/{id}", s.DeleteMessageHandler)
		// })
	})

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}

func (s *Server) llmHandler(w http.ResponseWriter, r *http.Request) {
	var req Message

	_ = json.NewDecoder(r.Body).Decode(&req)

	agent := core.Agent{
		SystemPrompt: "You are a helpful assistant.",
		ModelId:      core.GLM_4_5_AIR,
		MaxTokens:    2048,
		Temperature:  0.7,
	}

	// 1) Create MCP client over stdio
	ctx := r.Context()

	tr := transport.NewStdio("go", []string{"run", "./cmd/mcp/main.go"})
	cli := client.NewClient(tr)

	if err := cli.Start(ctx); err != nil {
		http.Error(w, fmt.Sprintf("failed to start MCP client: %v", err), http.StatusInternalServerError)
		return
	}
	defer cli.Close()

	// Initialize the MCP client
	_, err := cli.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize MCP client: %v", err), http.StatusInternalServerError)
		return
	}

	// 2) Fetch tools from MCP
	toolsResp, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list tools: %v", err), http.StatusInternalServerError)
		return
	}

	// 3) Map MCP tool descriptors to OpenAI function tools
	var oaTools []openai.Tool
	for _, t := range toolsResp.Tools {
		oaTools = append(oaTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // JSON Schema from MCP
			},
		})
	}
	agent.Tools = oaTools

	// 4) Call the agent
	response, err := agent.Invoke(req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error invoking agent: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResp, _ := json.Marshal(response)
	_, _ = w.Write(jsonResp)
}
