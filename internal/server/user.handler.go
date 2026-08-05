package server

import (
	"encoding/json"
	"net/http"
	"stockmind/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// POST /v1/users - Create a user
func (s *Server) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req database.CreateUserParams

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	user := &database.CreateUserParams{
		Name:     req.Name,
		Email:    req.Email,
		Provider: req.Provider,
	}

	if _, err := s.queries.CreateUser(r.Context(), *user); err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GET /v1/users - Get every user
func (s *Server) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := s.queries.GetUsers(r.Context())
	if err != nil {
		http.Error(w, "Failed to get users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GET /v1/users/{id} - Get one user
func (s *Server) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := s.queries.GetUserByID(r.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		http.Error(w, "User not found: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PUT /v1/users/{id} - Update a user's name and email
func (s *Server) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req database.UpdateUserParams
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	existingUser, err := s.queries.GetUserByID(r.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		http.Error(w, "User not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Absent fields keep the stored value, so a partial body is a partial update.
	user := &database.UpdateUserParams{
		ID:       uuid.Must(uuid.Parse(id)),
		Name:     existingUser.Name,
		Email:    existingUser.Email,
		Provider: existingUser.Provider,
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	if err := s.queries.UpdateUser(r.Context(), *user); err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// DELETE /v1/users/{id} - Delete a user
func (s *Server) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := s.queries.GetUserByID(r.Context(), uuid.Must(uuid.Parse(id))); err != nil {
		http.Error(w, "User not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if err := s.queries.DeleteUser(r.Context(), uuid.Must(uuid.Parse(id))); err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
