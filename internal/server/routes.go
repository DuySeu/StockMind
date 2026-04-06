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
		r.Get("/chat/stream", s.chatStreamHandler)

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

func parseChatRequest(r *http.Request) (string, uuid.UUID, []agent.Attachment, error) {
	var content string
	var sessionID uuid.UUID
	var attachments []agent.Attachment

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
			return "", uuid.Nil, nil, fmt.Errorf("invalid multipart form: %w", err)
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
			return "", uuid.Nil, nil, fmt.Errorf("invalid body: %w", err)
		}
		content = body.Content
		sessionID = body.SessionId
	}

	return content, sessionID, attachments, nil
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	content, sessionID, attachments, err := parseChatRequest(r)
	if err != nil {
		fmt.Printf("chatHandler parsing error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if content == "" && len(attachments) == 0 {
		http.Error(w, "content or attachment is required", http.StatusBadRequest)
		return
	}

	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	agentID := uuid.Must(uuid.Parse("01993ca8-a62e-79e3-995c-a46e25a4a2a4"))
	var sessionIdPtr *uuid.UUID
	if sessionID != uuid.Nil {
		sessionIdPtr = &sessionID
	}

	session, err := s.agent.GetOrCreateSession(&userID, &agentID, sessionIdPtr, &content)
	if err != nil {
		fmt.Printf("Failed to get or create session: %v\n", err)
		http.Error(w, "Failed to get or create session", http.StatusInternalServerError)
		return
	}

	// Initilize session
	err = session.Initialize()
	if err != nil {
		fmt.Printf("Failed to initialize session: %v\n", err)
		http.Error(w, "Failed to initialize session", http.StatusInternalServerError)
		return
	}

	stream := s.streamManager.CreateStream(session.GetSessionID())

	go func() {
		defer func() {
			stream.Close()
		}()

		session.AddChatCallback(func(event agent.ChatEvent) error {
			stream.AddEvent(event)
			return nil
		})

		err = session.HumanInput(content)
		if err != nil {
			fmt.Printf("Failed to send human input: %v\n", err)
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
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": session.GetSessionID().String(),
	})
}

func (s *Server) chatStreamHandler(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		http.Error(w, "invalid session_id format", http.StatusBadRequest)
		return
	}

	stream, exists := s.streamManager.GetStream(sessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if !exists {
		// Stream might be completed or never existed. Send a complete event to be safe.
		writeSSE(w, map[string]any{"type": "complete", "data": map[string]any{"session_id": sessionID.String()}})
		return
	}

	ch, history, isComplete := stream.Subscribe()
	if !isComplete {
		defer stream.Unsubscribe(ch)
	}

	// Replay history buffer
	inThinkingBlock := false
	for _, event := range history {
		sendEventToSSE(w, sessionID.String(), event, &inThinkingBlock)
	}

	if isComplete {
		return
	}

	// Listen for new events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			sendEventToSSE(w, sessionID.String(), event, &inThinkingBlock)
		}
	}
}

func sendEventToSSE(w http.ResponseWriter, sessionID string, event agent.ChatEvent, inThinkingBlock *bool) {
	switch event.Type {
	case agent.EventTypeText:
		if *inThinkingBlock {
			*inThinkingBlock = false
		}
		writeSSE(w, map[string]any{"type": "text_delta", "data": map[string]any{"text": event.Content}})
	case agent.EventTypeThinking:
		if !*inThinkingBlock {
			*inThinkingBlock = true
		}
		writeSSE(w, map[string]any{"type": "thinking_delta", "data": map[string]any{"thinking": event.Content}})
	case agent.EventTypeToolUse:
		writeSSE(w, map[string]any{"type": "tool_use", "data": map[string]any{"tool_calls": event.ToolUse}})
	case agent.EventTypeToolResult:
		// Map backend wrapper to frontend Message object
		var res map[string]any
		if event.ToolResult.OpenAI.Role != "" {
			res = map[string]any{
				"role": "tool",
				"content": []any{
					map[string]any{"type": "text", "text": event.ToolResult.OpenAI.Content},
				},
				"tool_call_id": event.ToolResult.OpenAI.ToolCallID,
			}
		} else if event.ToolResult.Anthropic.OfToolResult != nil {
			var content []any
			for _, part := range event.ToolResult.Anthropic.OfToolResult.Content {
				if part.OfText != nil {
					content = append(content, map[string]any{"type": "text", "text": part.OfText.Text})
				}
			}
			res = map[string]any{
				"role":         "tool",
				"content":      content,
				"tool_call_id": event.ToolResult.Anthropic.OfToolResult.ToolUseID,
			}
		} else {
			// Fallback
			res = map[string]any{"role": "tool", "content": []any{}}
		}
		writeSSE(w, map[string]any{"type": "tool_result", "data": map[string]any{"result": res}})
	}
	if event.IsEnd {
		writeSSE(w, map[string]any{"type": "complete", "data": map[string]any{"session_id": sessionID}})
	}
}

func writeSSE(w http.ResponseWriter, v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
