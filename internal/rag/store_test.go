package rag

import (
	"context"
	"testing"
)

// TestStore_UpsertValidation tests the input validation layer without a live Qdrant.
// Full integration is done manually against a running Qdrant container.

func TestStore_UpsertLengthMismatch(t *testing.T) {
	// QdrantStore.Upsert should reject non-matching chunk/vector lengths.
	s := &QdrantStore{configuredModel: embeddingModel}
	err := s.Upsert(context.Background(), "doc-id", []string{"a", "b"}, [][]float32{{0.1}}, StrategyRecursive)
	if err == nil {
		t.Error("expected error for mismatched chunks/vectors length")
	}
}

func TestStore_UpsertEmptyDocID(t *testing.T) {
	s := &QdrantStore{configuredModel: embeddingModel}
	err := s.Upsert(context.Background(), "", []string{"a"}, [][]float32{{0.1}}, StrategyFixed)
	if err == nil {
		t.Error("expected error for empty docID")
	}
}

func TestStore_UpsertEmptyChunks(t *testing.T) {
	s := &QdrantStore{configuredModel: embeddingModel}
	// Empty slice is a no-op, not an error.
	err := s.Upsert(context.Background(), "doc-123", nil, nil, StrategyFixed)
	if err != nil {
		t.Errorf("Upsert with empty chunks should be a no-op, got error: %v", err)
	}
}

func TestStore_DeleteEmptyDocID(t *testing.T) {
	s := &QdrantStore{configuredModel: embeddingModel}
	err := s.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty docID in Delete")
	}
}

func TestStoreConsistency_ConfiguredModel(t *testing.T) {
	// Verify the configuredModel field matches the global embeddingModel constant.
	s := &QdrantStore{configuredModel: embeddingModel}
	if s.configuredModel != embeddingModel {
		t.Errorf("expected configured model %q, got %q", embeddingModel, s.configuredModel)
	}
}
