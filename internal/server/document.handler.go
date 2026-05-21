package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"stockmind/internal/database"
	"stockmind/internal/service/worker"
)

// UploadDocumentHandler receives multipart form uploads with up to a 10MB limit.
func (s *Server) UploadDocumentHandler(w http.ResponseWriter, r *http.Request) {
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

	parts := strings.Split(header.Filename, ".")
	fileType := database.FileType("txt")
	if len(parts) > 1 {
		fileType = database.FileType(strings.ToLower(parts[len(parts)-1]))
	}
	if ft := r.FormValue("file_type"); ft != "" {
		fileType = database.FileType(ft)
	}

	doc, err := s.Upload(r.Context(), name, fileType, header.Size, file, database.ChunkingStrategySemantic)
	if err != nil {
		if err == worker.ErrQueueFull {
			http.Error(w, "server busy, try again later", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, fmt.Sprintf("failed to upload: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(doc)
}

// ListDocumentsHandler returns all indexed documents.
func (s *Server) ListDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	docs, err := s.queries.ListDocuments(r.Context())
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

	doc, err := s.queries.GetDocumentByID(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("document not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// DeleteDocumentHandler deletes a document and its embeddings.
func (s *Server) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid document ID", http.StatusBadRequest)
		return
	}

	if err := s.knowledgeStore.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete related vectors: %v", err), http.StatusInternalServerError)
		return
	}
	if err := s.objectStore.Delete(r.Context(), fmt.Sprintf("documents/%s/", id.String())); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete file from storage: %v", err), http.StatusInternalServerError)
		return
	}
	if err := s.queries.DeleteDocument(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete document metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) Upload(ctx context.Context, name string, fileType database.FileType, size int64, file io.Reader, strategy database.ChunkingStrategy) (database.Document, error) {
	docID := uuid.New()
	objectKey := fmt.Sprintf("documents/%s/%s", docID.String(), name)

	doc, err := s.queries.CreateDocument(ctx, database.CreateDocumentParams{
		ID:        docID,
		Name:      name,
		FileType:  fileType,
		SizeBytes: size,
		Strategy:  strategy,
	})
	if err != nil {
		return database.Document{}, fmt.Errorf("failed to save document metadata: %w", err)
	}

	if err := s.objectStore.Put(ctx, objectKey, file, size); err != nil {
		// Mark as failed since DB record exists but file upload failed
		s.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:     docID,
			Status: "failed",
		})
		return database.Document{}, fmt.Errorf("failed to upload file to storage: %w", err)
	}

	job := &worker.DocumentJob{
		DocID:     docID,
		Name:      name,
		FileType:  fileType,
		Strategy:  strategy,
		ObjectKey: objectKey,
	}

	if err := s.services.DocWorker.Enqueue(job); err != nil {
		return database.Document{}, err
	}

	return doc, nil
}
