package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"stockmind/internal/common"
	"stockmind/internal/database"
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

		// Documents
		r.Route("/documents", func(r chi.Router) {
			r.Post("/", s.UploadDocumentHandler)
			r.Get("/", s.ListDocumentsHandler)
			r.Get("/{id}", s.GetDocumentHandler)
			r.Delete("/{id}", s.DeleteDocumentHandler)
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

func parseChatRequest(r *http.Request) (string, uuid.UUID, []database.Attachment, error) {
	var content string
	var sessionID uuid.UUID
	var attachments []database.Attachment

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
					attachments = append(attachments, database.Attachment{
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
	var req Message
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	sessionID := req.SessionId
	if sessionID == uuid.Nil {
		userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
		id, err := s.queries.CreateSession(r.Context(), database.CreateSessionParams{
			ID:       uuid.New(),
			UserID:   userID,
			Title:    "New conversation",
			Metadata: []byte("{}"),
		})
		if err != nil {
			http.Error(w, "failed to create conversation: "+err.Error(), http.StatusInternalServerError)
			return
		}
		sessionID = id
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // prevent nginx/reverse-proxy buffering

	// HTTP/1.1: allow response bytes to flow while the handler still runs.
	if err := http.NewResponseController(w).EnableFullDuplex(); err != nil {
		// Best-effort; streaming still works on many stacks without this.
		log.Printf("sse: EnableFullDuplex: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	common.FlushSSE(w)

	// First SSE frame ASAP so clients see activity before DB + model work.
	common.WriteSSE(w, common.SSEEvent("start", map[string]any{"session_id": sessionID}))

	ctx := r.Context()
	eventCh, err := s.agent.Chat(ctx, sessionID, req.Content)
	if err != nil {
		common.WriteSSE(w, common.SSEEvent(common.EventError, map[string]any{"message": err.Error()}))
		return
	}

	for ev := range eventCh {
		switch ev.Type {
		case common.EventThinking:
			common.WriteSSE(w, common.SSEEvent(common.EventThinking, ev.Data))
		case common.EventText:
			common.WriteSSE(w, common.SSEEvent(common.EventText, ev.Content))
		case common.EventToolCall:
			common.WriteSSE(w, common.SSEEvent(common.EventToolCall, ev.Data))
		case common.EventToolResult:
			common.WriteSSE(w, common.SSEEvent(common.EventToolResult, ev.Data))
		case common.EventError:
			common.WriteSSE(w, common.SSEEvent(common.EventError, ev.Data))
		case common.EventDone:
			common.WriteSSE(w, common.SSEEvent(common.EventDone, map[string]any{"session_id": sessionID}))
		default:
			common.WriteSSE(w, common.SSEEvent(ev.Type, ev.Data))
		}
	}
}
