package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"stockmind/internal/rag"
)

// UploadDocumentHandler receives multipart form uploads with up to a 10MB limit.
// Expected fields: "file" (the document), "name" (optional), "strategy" (optional).
func (s *Server) UploadDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, fmt.Sprintf("File too large or invalid multipart form: %v", err), http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' field in multipart form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	strategyStr := r.FormValue("strategy")
	var strategy rag.Strategy
	switch strings.ToLower(strategyStr) {
	case "fixed":
		strategy = rag.StrategyFixed
	case "paragraph":
		strategy = rag.StrategyParagraph
	case "semantic":
		strategy = rag.StrategySemantic
	default:
		strategy = rag.StrategyRecursive
	}

	// Determine fileType by extension for parsing routing
	parts := strings.Split(header.Filename, ".")
	fileType := "txt" // fallback default
	if len(parts) > 1 {
		fileType = strings.ToLower(parts[len(parts)-1])
	}
	
	// Optional override from client
	if ft := r.FormValue("file_type"); ft != "" {
		fileType = ft
	}

	doc, err := s.documentService.Upload(r.Context(), name, fileType, header.Size, file, strategy)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted because processing will happen async
	json.NewEncoder(w).Encode(doc)
}

// ListDocumentsHandler returns all indexed documents.
func (s *Server) ListDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	docs, err := s.documentService.List(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": docs})
}

// GetDocumentHandler returns metadata for a single document by UUID.
func (s *Server) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	doc, err := s.documentService.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("document not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// DeleteDocumentHandler deletes a document and its embeddings from Postgres and Qdrant.
func (s *Server) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	err = s.documentService.Delete(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete document: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}
