package knowledge

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
	"google.golang.org/grpc/status"
)

const (
	CollectionName      = "stockmind"
	denseVectorName     = "dense"
	sparseVectorName    = "sparse"
	embeddingDimensions = 2048
)

// QdrantStore implements Store using the Qdrant gRPC client with named vectors.
type QdrantStore struct {
	points      pb.PointsClient
	collections pb.CollectionsClient
}

// NewQdrantStore creates a QdrantStore from an existing gRPC connection.
func NewQdrantStore(conn *grpc.ClientConn) *QdrantStore {
	return &QdrantStore{
		points:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
	}
}

// EnsureCollection creates the collection with named vectors if it doesn't exist.
func (s *QdrantStore) EnsureCollection(ctx context.Context) error {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(5, b)

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		_, reqErr := s.collections.Get(ctx, &pb.GetCollectionInfoRequest{
			CollectionName: CollectionName,
		})
		if reqErr != nil {
			if st, ok := status.FromError(reqErr); ok && st.Code() == codes.NotFound {
				return nil
			}
			return retry.RetryableError(reqErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("qdrant unavailable after retries: %w", err)
	}

	_, createErr := s.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: CollectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_ParamsMap{
				ParamsMap: &pb.VectorParamsMap{
					Map: map[string]*pb.VectorParams{
						denseVectorName: {
							Size:     embeddingDimensions,
							Distance: pb.Distance_Cosine,
						},
					},
				},
			},
		},
		SparseVectorsConfig: &pb.SparseVectorConfig{
			Map: map[string]*pb.SparseVectorParams{
				sparseVectorName: {},
			},
		},
	})
	if createErr != nil {
		if st, ok := status.FromError(createErr); !ok || st.Code() != codes.AlreadyExists {
			return fmt.Errorf("failed to create collection: %w", createErr)
		}
	}

	log.Printf("Qdrant collection ready: %s", CollectionName)
	return nil
}

// Upsert inserts or updates all chunks for a document with both dense and sparse vectors.
func (s *QdrantStore) Upsert(ctx context.Context, docID uuid.UUID, chunks []string, dense [][]float32, sparse []SparseVector) error {
	if len(chunks) != len(dense) || len(chunks) != len(sparse) {
		return errors.New("knowledge_base: chunks, dense, and sparse length mismatch")
	}
	if docID == uuid.Nil {
		return errors.New("knowledge_base: docID must not be nil")
	}
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*pb.PointStruct, 0, len(chunks))
	for i, text := range chunks {
		chunkID := uuid.NewSHA1(docID, []byte(fmt.Sprintf("%d", i)))
		points = append(points, &pb.PointStruct{
			Id: pb.NewIDUUID(chunkID.String()),
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vectors{
					Vectors: &pb.NamedVectors{
						Vectors: map[string]*pb.Vector{
							denseVectorName: {Data: dense[i]},
							sparseVectorName: {
								Indices: &pb.SparseIndices{Data: sparse[i].Indices},
								Data:    sparse[i].Values,
							},
						},
					},
				},
			},
			Payload: pb.NewValueMap(map[string]any{
				"doc_id":      docID.String(),
				"chunk_index": int64(i),
				"text":        text,
			}),
		})
	}

	_, err := s.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("knowledge_base: upsert failed for doc %s: %w", docID, err)
	}
	return nil
}

// Delete removes all points associated with docID.
func (s *QdrantStore) Delete(ctx context.Context, docID uuid.UUID) error {
	if docID == uuid.Nil {
		return errors.New("knowledge_base: docID must not be nil")
	}
	_, err := s.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						pb.NewMatch("doc_id", docID.String()),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("knowledge_base: delete failed for doc %s: %w", docID, err)
	}
	return nil
}

// SearchDense performs semantic search using the dense vector.
func (s *QdrantStore) SearchDense(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error) {
	resp, err := s.points.Search(ctx, &pb.SearchPoints{
		CollectionName: CollectionName,
		Vector:         vector,
		VectorName:     pb.PtrOf(denseVectorName),
		Limit:          uint64(topK),
		WithPayload:    pb.NewWithPayload(true),
		ScoreThreshold: &threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("knowledge_base: dense search failed: %w", err)
	}
	return pointsToResults(resp.Result), nil
}

// SearchSparse performs keyword search using the sparse vector.
func (s *QdrantStore) SearchSparse(ctx context.Context, vector SparseVector, topK int) ([]SearchResult, error) {
	resp, err := s.points.Query(ctx, &pb.QueryPoints{
		CollectionName: CollectionName,
		Query:          pb.NewQuerySparse(vector.Indices, vector.Values),
		Using:          pb.PtrOf(sparseVectorName),
		Limit:          pb.PtrOf(uint64(topK)),
		WithPayload:    pb.NewWithPayloadInclude("text", "doc_id", "chunk_index"),
	})
	if err != nil {
		return nil, fmt.Errorf("knowledge_base: sparse search failed: %w", err)
	}
	return scoredPointsToResults(resp.Result), nil
}

// SearchHybrid performs hybrid search using RRF fusion of dense + sparse.
func (s *QdrantStore) SearchHybrid(ctx context.Context, dense []float32, sparse SparseVector, topK int) ([]SearchResult, error) {
	resp, err := s.points.Query(ctx, &pb.QueryPoints{
		CollectionName: CollectionName,
		Prefetch: []*pb.PrefetchQuery{
			{
				Query: pb.NewQueryDense(dense),
				Using: pb.PtrOf(denseVectorName),
				Limit: pb.PtrOf(uint64(topK * 4)),
			},
			{
				Query: pb.NewQuerySparse(sparse.Indices, sparse.Values),
				Using: pb.PtrOf(sparseVectorName),
				Limit: pb.PtrOf(uint64(topK * 4)),
			},
		},
		Query:       pb.NewQueryFusion(pb.Fusion_RRF),
		Limit:       pb.PtrOf(uint64(topK)),
		WithPayload: pb.NewWithPayloadInclude("text", "doc_id", "chunk_index"),
	})
	if err != nil {
		return nil, fmt.Errorf("knowledge_base: hybrid search failed: %w", err)
	}
	return scoredPointsToResults(resp.Result), nil
}

func pointsToResults(points []*pb.ScoredPoint) []SearchResult {
	results := make([]SearchResult, 0, len(points))
	for _, p := range points {
		results = append(results, SearchResult{
			Text:       payloadString(p.Payload, "text"),
			Score:      p.Score,
			DocID:      payloadString(p.Payload, "doc_id"),
			ChunkIndex: int(payloadInt(p.Payload, "chunk_index")),
		})
	}
	return results
}

func scoredPointsToResults(points []*pb.ScoredPoint) []SearchResult {
	return pointsToResults(points)
}

func payloadString(payload map[string]*pb.Value, key string) string {
	if v, ok := payload[key]; ok {
		return v.GetStringValue()
	}
	return ""
}

func payloadInt(payload map[string]*pb.Value, key string) int64 {
	if v, ok := payload[key]; ok {
		return v.GetIntegerValue()
	}
	return 0
}
