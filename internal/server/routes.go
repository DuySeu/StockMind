package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"stockmind/internal/agent"
)

type Message struct {
	Content   string    `json:"content"`
	SessionId uuid.UUID `json:"session_id"`
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

	// SPA
	r.Handle("/*", spaHandler())

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Websocket
		// r.Get("/ws", s.websocketHandler)
		r.Post("/chat", s.chatHandler)

		// Users
		r.Route("/users", func(r chi.Router) {
			r.Post("/", s.CreateUserHandler)
			r.Get("/", s.GetUsersHandler)
			r.Get("/{id}", s.GetUserByIDHandler)
			r.Put("/{id}", s.UpdateUserHandler)
			r.Delete("/{id}", s.DeleteUserHandler)
		})

		// Sessions
		r.Route("/sessions", func(r chi.Router) {
			r.Get("/", s.GetSessionsHandler)
			r.Get("/{id}", s.GetMessagesBySessionIdHandler)
			// r.Post("/", s.CreateSessionHandler)
			// r.Put("/{id}", s.UpdateSessionHandler)
			r.Delete("/{id}", s.DeleteSessionHandler)
		})

		// Messages
		r.Route("/messages", func(r chi.Router) {
			// r.Post("/", s.CreateMessageHandler)
			// r.Get("/", s.GetMessagesHandler)
			// r.Get("/{id}", s.GetMessageByIDHandler)
			// r.Put("/{id}", s.UpdateMessageHandler)
			// r.Delete("/{id}", s.DeleteMessageHandler)
		})

		// Agent Flows
		r.Route("/agent_flows", func(r chi.Router) {
			r.Get("/", s.ListAgentFlowsHandler)
			// r.Get("/{id}", s.GetAgentFlowByIDHandler)
			r.Post("/", s.CreateAgentFlowHandler)
			// r.Put("/{id}", s.UpdateAgentFlowHandler)
			// r.Delete("/{id}", s.DeleteAgentFlowHandler)
		})

		// Stock
		r.Route("/stock", func(r chi.Router) {
			r.Get("/price-board", s.GetPriceBoardHandler)
			r.Get("/watchlist", s.GetWatchlistHandler)
			r.Get("/research-reports", s.GetResearchReportsHandler)
			r.Get("/research-reports/{id}", s.GetResearchReportByIDHandler)
			r.Post("/add-symbol", s.AddSymbolInPriceBoardHandler)
			r.Post("/research", s.MarketResearchHandler)
			r.Post("/research/stream", s.MarketResearchStreamHandler)
		})

		r.Route("/news", func(r chi.Router) {
			r.Get("/", s.GetNewsHandler)
		})
	})

	return r
}

func spaHandler() http.HandlerFunc {
	spaFS := os.DirFS("frontend/dist")
	return func(w http.ResponseWriter, r *http.Request) {
		// Any path not ending with a file extension is served as index.html
		if path.Ext(r.URL.Path) == "" || r.URL.Path == "/" {
			http.ServeFileFS(w, r, spaFS, "index.html")
			return
		}
		fmt.Println("Serving file", "path", path.Clean(r.URL.Path))
		f, err := spaFS.Open(strings.TrimPrefix(path.Clean(r.URL.Path), "/"))
		if err == nil {
			defer f.Close()
		}
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Content not found"))
			return
		}
		http.FileServer(http.FS(spaFS)).ServeHTTP(w, r)
	}
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var content string
	var sessionID uuid.UUID
	var attachments []agent.Attachment

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
			fmt.Println("invalid multipart form:", err)
			return
		}
		content = r.FormValue("content")
		sessionIDStr := r.FormValue("session_id")
		if sessionIDStr != "" {
			if id, err := uuid.Parse(sessionIDStr); err == nil {
				sessionID = id
			}
		}

		// Handle files
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			for _, fileHeaders := range r.MultipartForm.File {
				for _, fileHeader := range fileHeaders {
					f, err := fileHeader.Open()
					if err != nil {
						fmt.Println("failed to open file:", err)
						continue
					}
					data, err := io.ReadAll(f)
					f.Close()
					if err != nil {
						fmt.Println("failed to read file:", err)
						continue
					}
					attachments = append(attachments, agent.Attachment{
						Name:      fileHeader.Filename,
						MediaType: fileHeader.Header.Get("Content-Type"),
						Data:      data,
					})
				}
			}
		}
	} else {
		// Parse JSON body
		var body Message
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			fmt.Println("invalid body:", err)
			return
		}
		content = body.Content
		sessionID = body.SessionId
	}

	if content == "" && len(attachments) == 0 {
		fmt.Println("content or attachment is required")
		return
	}

	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	agentID := uuid.Must(uuid.Parse("01993ca8-a62e-79e3-995c-a46e25a4a2a2"))
	var sessionIdPtr *uuid.UUID
	if sessionID != uuid.Nil {
		sessionIdPtr = &sessionID
	}

	session, err := s.agent.GetOrCreateSession(&userID, &agentID, sessionIdPtr, &content)
	if err != nil {
		fmt.Println("Failed to get or create session", "error", err)
		return
	}
	// Initilize session
	err = session.Initialize()
	if err != nil {
		fmt.Println("Failed to initialize session", "error", err)
		return
	}
	inThinkingBlock := false
	session.AddChatCallback(func(event agent.ChatEvent) error {
		switch event.Type {
		case agent.EventTypeText:
			if inThinkingBlock {
				inThinkingBlock = false
			}
			writeSSE(w, map[string]any{"type": "text_delta", "data": map[string]any{"text": event.Content}})
		case agent.EventTypeThinking:
			if !inThinkingBlock {
				inThinkingBlock = true
			}
			writeSSE(w, map[string]any{"type": "thinking_delta", "data": map[string]any{"thinking": event.Content}})
		case agent.EventTypeToolUse:
			writeSSE(w, map[string]any{"type": "tool_use", "data": map[string]any{"tool_calls": event.ToolUse}})
		case agent.EventTypeToolResult:
			writeSSE(w, map[string]any{"type": "tool_result", "data": map[string]any{"result": event.ToolResult}})
		}
		if event.IsEnd {
			writeSSE(w, map[string]any{"type": "complete", "data": map[string]any{"session_id": session.SessionID().String()}})
		}
		return nil
	})

	err = session.HumanInput(content, attachments)
	if err != nil {
		fmt.Println("Failed to send human input", "error", err)
		return
	}
	nLoop := 0
	fmt.Println("Starting agent execution loop")
	for !session.IsHumanTurn() {
		fmt.Printf("Loop iteration %d: continuing turn...\n", nLoop+1)
		err = session.ContinueTurn()
		if err != nil {
			fmt.Printf("Failed to continue turn (iteration: %d, error: %v)\n", nLoop+1, err)
			return
		}
		fmt.Printf("Loop iteration %d: completed successfully\n", nLoop+1)
		nLoop++
		if nLoop > 10 {
			fmt.Println("Too many loops (max: 10), something is wrong")
			return
		}
	}
	fmt.Printf("Agent execution loop completed after %d iterations\n", nLoop)
}

func writeSSE(w http.ResponseWriter, v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
