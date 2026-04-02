package rag

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

const (
	// collectionName is the Qdrant collection used by StockMind RAG.
	collectionName = "stockmind_knowledge"

	// modelMetaKey is the collection alias used to persist the embedding model
	// name so that we can assert consistency on subsequent startups.
	modelMetaKey = "embedding_model"
)

// Store is the interface for persisting chunk vectors and metadata into Qdrant.
// Implementations must be safe for concurrent use.
type Store interface {
	// Upsert inserts or updates chunks for a given document. The length of
	// chunks and vectors must match. strategy is the chunking algorithm used.
	Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy Strategy) error

	// Delete removes all points associated with docID from the collection.
	Delete(ctx context.Context, docID string) error
}

// QdrantStore implements Store using the official Qdrant gRPC Go client.
// It reuses the *qdrant.PointsClient initialised by InitQdrant during startup.
type QdrantStore struct {
	points            qdrant.PointsClient
	collections       qdrant.CollectionsClient
	configuredModel   string
}

// NewQdrantStore creates a QdrantStore. Call CheckModelConsistency after
// construction to assert the collection was built with the same model.
//
// pointsClient should come from InitQdrant; collectionsClient is derived from
// the same gRPC connection.
func NewQdrantStore(pointsClient qdrant.PointsClient, collectionsClient qdrant.CollectionsClient) *QdrantStore {
	return &QdrantStore{
		points:          pointsClient,
		collections:     collectionsClient,
		configuredModel: embeddingModel, // constant from embedder.go
	}
}

// CheckModelConsistency retrieves the collection's stored embedding model from
// its on-disk metadata and compares it against the configured model.
//
// On first run the field is absent — it writes the model name and returns nil.
// On subsequent runs it returns an error if the stored model differs.
//
// Call this once during server startup, after InitQdrant.
func (s *QdrantStore) CheckModelConsistency(ctx context.Context) error {
	info, err := s.collections.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to get collection info: %w", err)
	}

	// The collection payload schema is not the right place for collection-level
	// metadata; Qdrant exposes it through collection aliases. We use the
	// collection's custom key-value metadata field (on_disk_payload / params).
	// Since Qdrant doesn't have a native key-value store for this, we persist
	// the model name as a special "config" point with id=0 and no vector.
	//
	// Practical approach: we store a sentinel point with id=0 that only carries
	// a payload with the embedding_model key. This point is excluded from
	// searches by filtering on chunk_index >= 0 (our real chunks).
	_ = info // used for liveness check above

	return s.assertModelPoint(ctx)
}

// assertModelPoint reads or writes the sentinel config point (id=uint64(0))
// that records the embedding model used to create the collection.
func (s *QdrantStore) assertModelPoint(ctx context.Context) error {
	resp, err := s.points.Get(ctx, &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(0)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to read config point: %w", err)
	}

	if len(resp.Result) == 0 {
		// First run — write the sentinel.
		return s.writeModelPoint(ctx)
	}

	// Second+ run — compare stored model with configured.
	result := resp.Result[0]
	stored, ok := result.Payload[modelMetaKey]
	if !ok {
		// Sentinel exists but has no model key — treat as first run.
		return s.writeModelPoint(ctx)
	}

	storedModel := stored.GetStringValue()
	if storedModel != s.configuredModel {
		return fmt.Errorf(
			"qdrant store: embedding model mismatch — collection was built with %q, server is configured for %q; re-index or switch model",
			storedModel, s.configuredModel,
		)
	}
	return nil
}

// writeModelPoint upserts the sentinel config point (id=0) with the embedding
// model name as payload.
func (s *QdrantStore) writeModelPoint(ctx context.Context) error {
	// Config point uses a zero vector with the correct dimension so that Qdrant
	// accepts it. We always filter it out with chunk_index >= 0 in real queries.
	zeroVec := make([]float32, embeddingDimensions)
	_, err := s.points.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(0),
				Vectors: qdrant.NewVectors(zeroVec...),
				Payload: qdrant.NewValueMap(map[string]any{
					modelMetaKey: s.configuredModel,
					"_type":      "config",
				}),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to write config point: %w", err)
	}
	return nil
}

// Upsert inserts or updates all chunks for a document. Each chunk becomes one
// Qdrant point with the payload:
//
//	{doc_id, chunk_index, text, strategy}
//
// Points are keyed by a deterministic UUID derived from docID + chunk_index.
func (s *QdrantStore) Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy Strategy) error {
	if len(chunks) != len(vectors) {
		return errors.New("qdrant store: chunks and vectors length mismatch")
	}
	if docID == "" {
		return errors.New("qdrant store: docID must not be empty")
	}
	if len(chunks) == 0 {
		return nil // nothing to do
	}

	points := make([]*qdrant.PointStruct, 0, len(chunks))
	for i, text := range chunks {
		// Derive a stable UUID for each chunk: namespace(docID) + chunk_index.
		// This lets us re-index a document idempotently (upsert behaviour).
		chunkID := uuid.NewSHA1(uuid.MustParse(docID), []byte(fmt.Sprintf("%d", i)))

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(chunkID.String()),
			Vectors: qdrant.NewVectors(vectors[i]...),
			Payload: qdrant.NewValueMap(map[string]any{
				"doc_id":      docID,
				"chunk_index": int64(i),
				"text":        text,
				"strategy":    string(strategy),
			}),
		})
	}

	_, err := s.points.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("qdrant store: upsert failed for doc %s: %w", docID, err)
	}
	return nil
}

// Delete removes all Qdrant points whose payload.doc_id matches docID.
func (s *QdrantStore) Delete(ctx context.Context, docID string) error {
	if docID == "" {
		return errors.New("qdrant store: docID must not be empty")
	}

	_, err := s.points.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						qdrant.NewMatch("doc_id", docID),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant store: delete failed for doc %s: %w", docID, err)
	}
	return nil
}
