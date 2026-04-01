package rag

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/sethvargo/go-retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// InitQdrant establishes a connection to Qdrant with fibonacci retry backoff.
// It creates the "stockmind_knowledge" collection with 2048 dimensions if not present.
func InitQdrant(ctx context.Context, host, port string) (*qdrant.PointsClient, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	var conn *grpc.ClientConn
	var err error

	conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc client: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)

	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithMaxRetries(5, b)

	err = retry.Do(ctx, b, func(ctx context.Context) error {
		_, reqErr := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
			CollectionName: "stockmind_knowledge",
		})
		
		if reqErr != nil {
			if st, ok := status.FromError(reqErr); ok && st.Code() == codes.NotFound {
				// We connected, but collection is missing. This is fine, we will create it.
				return nil
			}
			// Connection error or Qdrant isn't ready
			log.Printf("Qdrant check failed: %v", reqErr)
			return retry.RetryableError(reqErr)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("qdrant unavailable after retries: %w", err)
	}

	// Create the collection idempotently
	_, createErr := collectionsClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: "stockmind_knowledge",
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     2048,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	
	// Ignore AlreadyExists errors
	if createErr != nil {
		if st, ok := status.FromError(createErr); !ok || st.Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("failed to create stockmind_knowledge collection: %w", createErr)
		}
	}

	log.Printf("Qdrant collection ready: stockmind_knowledge")
	pointsClient := qdrant.NewPointsClient(conn)
	return &pointsClient, nil
}
