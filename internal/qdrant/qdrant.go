package qdrant

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/sethvargo/go-retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	CollectionName      = "stockmind_knowledge"
	modelMetaKey        = "embedding_model"
	embeddingDimensions = 2048
)

// SearchResult represents a single search hit from Qdrant.
type SearchResult struct {
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
	DocID      string  `json:"doc_id"`
	ChunkIndex int     `json:"chunk_index"`
}

// Store is the interface for persisting and querying chunk vectors in Qdrant.
type Store interface {
	Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy string) error
	Delete(ctx context.Context, docID string) error
	Search(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error)
}

// InitQdrant establishes a gRPC connection to Qdrant with fibonacci retry backoff
// and creates the collection if it does not exist.
func InitQdrant(ctx context.Context, host string, port int) (*grpc.ClientConn, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc client: %w", err)
	}

	collectionsClient := pb.NewCollectionsClient(conn)

	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(5, b)

	err = retry.Do(ctx, b, func(ctx context.Context) error {
		_, reqErr := collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
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

	_, createErr := collectionsClient.Create(ctx, &pb.CreateCollection{
		CollectionName: CollectionName,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     embeddingDimensions,
			Distance: pb.Distance_Cosine,
		}),
	})
	if createErr != nil {
		if st, ok := status.FromError(createErr); !ok || st.Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("failed to create collection: %w", createErr)
		}
	}

	log.Printf("Qdrant collection ready: %s", CollectionName)
	return conn, nil
}

// QdrantStore implements Store using the official Qdrant gRPC client.
type QdrantStore struct {
	points          pb.PointsClient
	collections     pb.CollectionsClient
	configuredModel string
}

// NewQdrantStore creates a QdrantStore from an existing gRPC connection.
func NewQdrantStore(conn *grpc.ClientConn, embeddingModel string) *QdrantStore {
	return &QdrantStore{
		points:          pb.NewPointsClient(conn),
		collections:     pb.NewCollectionsClient(conn),
		configuredModel: embeddingModel,
	}
}

// CheckModelConsistency verifies the collection was built with the same embedding model.
func (s *QdrantStore) CheckModelConsistency(ctx context.Context) error {
	resp, err := s.points.Get(ctx, &pb.GetPoints{
		CollectionName: CollectionName,
		Ids:            []*pb.PointId{pb.NewIDNum(0)},
		WithPayload:    pb.NewWithPayload(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant store: failed to read config point: %w", err)
	}

	if len(resp.Result) == 0 {
		return s.writeModelPoint(ctx)
	}

	stored, ok := resp.Result[0].Payload[modelMetaKey]
	if !ok {
		return s.writeModelPoint(ctx)
	}

	if storedModel := stored.GetStringValue(); storedModel != s.configuredModel {
		return fmt.Errorf(
			"qdrant store: embedding model mismatch — collection built with %q, configured %q",
			storedModel, s.configuredModel,
		)
	}
	return nil
}

func (s *QdrantStore) writeModelPoint(ctx context.Context) error {
	zeroVec := make([]float32, embeddingDimensions)
	_, err := s.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Points: []*pb.PointStruct{
			{
				Id:      pb.NewIDNum(0),
				Vectors: pb.NewVectors(zeroVec...),
				Payload: pb.NewValueMap(map[string]any{
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
func (s *QdrantStore) Upsert(ctx context.Context, docID string, chunks []string, vectors [][]float32, strategy string) error {
	if len(chunks) != len(vectors) {
		return errors.New("qdrant store: chunks and vectors length mismatch")
	}
	if docID == "" {
		return errors.New("qdrant store: docID must not be empty")
	}
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*pb.PointStruct, 0, len(chunks))
	for i, text := range chunks {
		chunkID := uuid.NewSHA1(uuid.MustParse(docID), []byte(fmt.Sprintf("%d", i)))
		points = append(points, &pb.PointStruct{
			Id:      pb.NewIDUUID(chunkID.String()),
			Vectors: pb.NewVectors(vectors[i]...),
			Payload: pb.NewValueMap(map[string]any{
				"doc_id":      docID,
				"chunk_index": int64(i),
				"text":        text,
				"strategy":    strategy,
			}),
		})
	}

	_, err := s.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("qdrant store: upsert failed for doc %s: %w", docID, err)
	}
	return nil
}

// Delete removes all points associated with docID.
func (s *QdrantStore) Delete(ctx context.Context, docID string) error {
	if docID == "" {
		return errors.New("qdrant store: docID must not be empty")
	}

	_, err := s.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						pb.NewMatch("doc_id", docID),
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

// Search returns the most similar chunks to the given vector.
func (s *QdrantStore) Search(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error) {
	if len(vector) == 0 {
		return nil, errors.New("qdrant store: query vector must not be empty")
	}
	if topK <= 0 {
		return nil, errors.New("qdrant store: topK must be positive")
	}

	resp, err := s.points.Search(ctx, &pb.SearchPoints{
		CollectionName: CollectionName,
		Vector:         vector,
		Limit:          uint64(topK),
		WithPayload:    pb.NewWithPayload(true),
		ScoreThreshold: &threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant store: search failed: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Result))
	for _, point := range resp.Result {
		payload := point.Payload
		var textStr, docIDStr string
		var chunkIndex int
		if v, ok := payload["text"]; ok {
			textStr = v.GetStringValue()
		}
		if v, ok := payload["doc_id"]; ok {
			docIDStr = v.GetStringValue()
		}
		if v, ok := payload["chunk_index"]; ok {
			chunkIndex = int(v.GetIntegerValue())
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
