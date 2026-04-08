package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"stockmind/internal/database"
	"stockmind/internal/rag"
)

// DocumentService provides business logic for managing knowledge base documents.
type DocumentService struct {
	db     *database.Queries
	worker *rag.Worker
	store  rag.Store
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(db *database.Queries, worker *rag.Worker, store rag.Store) *DocumentService {
	return &DocumentService{
		db:     db,
		worker: worker,
		store:  store,
	}
}

// Upload handles receiving a new document, verifying basic constraints, and adding it to the processing queue.
func (s *DocumentService) Upload(ctx context.Context, name, fileType string, size int64, file io.Reader, strategy rag.Strategy) (database.Document, error) {
	// Create a temporary file to store the upload temporarily
	tempFile, err := os.CreateTemp("", fmt.Sprintf("upload-*-%s.%s", uuid.New().String(), fileType))
	if err != nil {
		return database.Document{}, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Ensure we clean up the file if something fails before we enqueue it
	cleanupOnFail := true
	defer func() {
		if cleanupOnFail {
			os.Remove(tempFile.Name())
		}
	}()

	// Save the file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return database.Document{}, fmt.Errorf("failed to save uploaded file: %w", err)
	}

	docID := uuid.New()

	// Insert tracking record into database
	doc, err := s.db.CreateDocument(ctx, database.CreateDocumentParams{
		ID:        docID,
		Name:      name,
		FileType:  fileType,
		SizeBytes: size,
		Strategy:  string(strategy),
	})
	if err != nil {
		return database.Document{}, fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Create and enqueue the background processing job
	job := &rag.Job{
		DocID:    docID,
		Name:     name,
		FileType: fileType,
		Strategy: strategy,
		TempFile: tempFile.Name(), // the background worker will delete this upon completion
	}
	
	// The job is now responsible for the file
	cleanupOnFail = false
	s.worker.Enqueue(job)

	return doc, nil
}

// List returns all registered documents.
func (s *DocumentService) List(ctx context.Context) ([]database.Document, error) {
	return s.db.ListDocuments(ctx)
}

// GetByID returns the details of a single document by its UUID.
func (s *DocumentService) GetByID(ctx context.Context, id uuid.UUID) (database.Document, error) {
	return s.db.GetDocumentByID(ctx, id)
}

// Delete removes the document from the PostgreSQL database and its indexed chunks from Qdrant.
func (s *DocumentService) Delete(ctx context.Context, id uuid.UUID) error {
	// First delete from Qdrant. It's safer to orphan vectors in Qdrant (if DB delete fails) 
	// than to orphan a DB record with deleted vectors.
	err := s.store.Delete(ctx, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete related vectors from qdrant: %w", err)
	}

	// Delete from PostgreSQL
	err = s.db.DeleteDocument(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete document metadata: %w", err)
	}

	return nil
}
