package knowledge_base

import (
	"context"
	"errors"
	"fmt"
	"log"

	"stockmind/internal/common"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SearchMode determines which search strategy to use.
type SearchMode string

const (
	SearchSemantic SearchMode = "semantic"
	SearchKeyword  SearchMode = "keyword"
	SearchHybrid   SearchMode = "hybrid"
)

// SearchResult represents a single retrieval hit.
type SearchResult struct {
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
	DocID      string  `json:"doc_id"`
	ChunkIndex int     `json:"chunk_index"`
}

// SparseVector represents a BM25-generated sparse vector.
type SparseVector struct {
	Indices []uint32
	Values  []float32
}

var (
	ErrEmptyQuery   = errors.New("query must not be empty")
	ErrNoResults    = errors.New("no relevant results found")
	ErrStoreUnavail = errors.New("vector store unavailable")
	ErrEmbedFailed  = errors.New("embedding generation failed")
)

// Retriever is the main entry point for querying the knowledge base.
type Retriever interface {
	Search(ctx context.Context, query string, mode SearchMode, topK int) ([]SearchResult, error)
}

// Store abstracts the vector database operations.
type Store interface {
	Upsert(ctx context.Context, docID uuid.UUID, chunks []string, dense [][]float32, sparse []SparseVector) error
	Delete(ctx context.Context, docID uuid.UUID) error
	SearchDense(ctx context.Context, vector []float32, topK int, threshold float32) ([]SearchResult, error)
	SearchSparse(ctx context.Context, vector SparseVector, topK int) ([]SearchResult, error)
	SearchHybrid(ctx context.Context, dense []float32, sparse SparseVector, topK int) ([]SearchResult, error)
}

// Embedder converts text into dense embedding vectors.
type Embedder interface {
	Embed(ctx context.Context, input []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
	Dimensions() int
}

// KnowledgeBase is the top-level container returned by New().
type KnowledgeBase struct {
	Retriever Retriever
	Pipeline  *IngestPipeline
	Store     Store
}

// New initializes the full knowledge base: store, embedder, BM25, retriever, and pipeline.
func New(ctx context.Context, cfg *common.Config, dbPool *pgxpool.Pool) (*KnowledgeBase, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Qdrant.Host, cfg.Qdrant.Port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("knowledge_base: failed to connect qdrant: %w", err)
	}

	store := NewQdrantStore(conn)
	if err := store.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("knowledge_base: %w", err)
	}
	log.Println("Knowledge base Qdrant collection ready")

	embedder, err := NewEmbedService(0, cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("knowledge_base: %w", err)
	}

	tokenizer := NewTokenizer(30000)

	retriever := &retriever{
		embedder:  embedder,
		tokenizer: tokenizer,
		store:     store,
	}

	pipeline := &IngestPipeline{
		embedder:  embedder,
		tokenizer: tokenizer,
		store:     store,
	}

	return &KnowledgeBase{
		Retriever: retriever,
		Pipeline:  pipeline,
		Store:     store,
	}, nil
}
