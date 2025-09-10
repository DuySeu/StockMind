package server

import (
	"encoding/json"
	"log"
	"net/http"

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

	r.Get("/", s.HelloWorldHandler)
	r.Get("/health", s.healthHandler)

	r.Route("/v1", func(r chi.Router) {
		// Websocket
		r.Get("/ws", s.websocketHandler)

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
