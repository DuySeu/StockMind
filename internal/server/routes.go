package server

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

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
		r.Post("/chat", s.ChatHandler)

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
			r.Post("/fundamental-analysis", s.FundamentalAnalysisHandler)
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
		slog.Debug("serving file", "path", path.Clean(r.URL.Path))
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
