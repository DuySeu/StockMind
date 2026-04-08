package rag

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sethvargo/go-retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	// CollectionName is the Qdrant collection used by StockMind RAG.
	CollectionName = "stockmind_knowledge"

	// modelMetaKey is the collection alias used to persist the embedding model
	// name so that we can assert consistency on subsequent startups.
	modelMetaKey = "embedding_model"
)

// ---- Connection & Initialization ----

// InitQdrant establishes a connection to Qdrant with fibonacci retry backoff.
// It creates the "stockmind_knowledge" collection with 2048 dimensions if not present.
func InitQdrant(ctx context.Context, host, port string) (*grpc.ClientConn, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc client: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)

	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(5, b)

	err = retry.Do(ctx, b, func(ctx context.Context) error {
		_, reqErr := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
			CollectionName: CollectionName,
		})
		
		if reqErr != nil {
			if st, ok := status.FromError(reqErr); ok && st.Code() == codes.NotFound {
				return nil
			}
			log.Printf("Qdrant check failed: %v", reqErr)
			return retry.RetryableError(reqErr)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("qdrant unavailable after retries: %w", err)
	}

	_, createErr := collectionsClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: CollectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     2048,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	
	if createErr != nil {
		if st, ok := status.FromError(createErr); !ok || st.Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("failed to create stockmind_knowledge collection: %w", createErr)
		}
	}

	log.Printf("Qdrant collection ready: %s", CollectionName)
	return conn, nil
}

type SearchResult struct {
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
	DocID      string  `json:"doc_id"`
	ChunkIndex int     `json:"chunk_index"`
}

// Store is the interface for persisting chunk vectors and metadata into Qdrant.
// Implementations must be safe for concurrent use.
type Store interface {
	// Upsert inserts or updates chunks for a given document. The length of
	// chunks and vectors must match. strategy is the chunking algorithm used.
	Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy Strategy) error

	// Delete removes all points associated with docID from the collection.
	Delete(ctx context.Context, docID string) error

	// Search returns the most similar chunks to the given vector, filtered by a minimum score threshold.
	Search(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error)
}

// QdrantStore implements Store using the official Qdrant gRPC Go client.
type QdrantStore struct {
	points            qdrant.PointsClient
	collections       qdrant.CollectionsClient
	configuredModel   string
}

// NewQdrantStore creates a QdrantStore. Call CheckModelConsistency after
// construction to assert the collection was built with the same model.
func NewQdrantStore(conn *grpc.ClientConn) *QdrantStore {
	return &QdrantStore{
		points:          qdrant.NewPointsClient(conn),
		collections:     qdrant.NewCollectionsClient(conn),
		configuredModel: embeddingModel, // constant from embedder.go
	}
}

// CheckModelConsistency retrieves the collection's stored embedding model from
// its on-disk metadata and compares it against the configured model.
func (s *QdrantStore) CheckModelConsistency(ctx context.Context) error {
	_, err := s.collections.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: CollectionName,
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to get collection info: %w", err)
	}

	return s.assertModelPoint(ctx)
}

func (s *QdrantStore) assertModelPoint(ctx context.Context) error {
	resp, err := s.points.Get(ctx, &qdrant.GetPoints{
		CollectionName: CollectionName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(0)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to read config point: %w", err)
	}

	if len(resp.Result) == 0 {
		return s.writeModelPoint(ctx)
	}

	result := resp.Result[0]
	stored, ok := result.Payload[modelMetaKey]
	if !ok {
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

func (s *QdrantStore) writeModelPoint(ctx context.Context) error {
	zeroVec := make([]float32, embeddingDimensions)
	_, err := s.points.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: CollectionName,
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

// Upsert inserts or updates all chunks for a document.
func (s *QdrantStore) Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy Strategy) error {
	if len(chunks) != len(vectors) {
		return errors.New("qdrant store: chunks and vectors length mismatch")
	}
	if docID == "" {
		return errors.New("qdrant store: docID must not be empty")
	}
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(chunks))
	for i, text := range chunks {
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
		CollectionName: CollectionName,
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
		CollectionName: CollectionName,
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

// Search searches for the most similar chunks to the query vector.
func (s *QdrantStore) Search(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error) {
	if len(vector) == 0 {
		return nil, errors.New("qdrant store: query vector must not be empty")
	}
	if topK <= 0 {
		return nil, errors.New("qdrant store: topK must be positive")
	}

	searchReq := &qdrant.SearchPoints{
		CollectionName: CollectionName,
		Vector:         vector,
		Limit:          uint64(topK),
		WithPayload:    qdrant.NewWithPayload(true),
		ScoreThreshold: &threshold,
	}

	resp, err := s.points.Search(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("qdrant store: search failed: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Result))
	for _, point := range resp.Result {
		payload := point.Payload
		
		textStr := ""
		if textVal, ok := payload["text"]; ok {
			textStr = textVal.GetStringValue()
		}
		
		docIDStr := ""
		if docIDVal, ok := payload["doc_id"]; ok {
			docIDStr = docIDVal.GetStringValue()
		}
		
		var chunkIndex int = 0
		if idxVal, ok := payload["chunk_index"]; ok {
			chunkIndex = int(idxVal.GetIntegerValue())
		}

		results = append(results, SearchResult{
			Text:       textStr,
			Score:      point.Score,
			DocID:      docIDStr,
			ChunkIndex: chunkIndex,
		})
	}

	return results, nil
}

