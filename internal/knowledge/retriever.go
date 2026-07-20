package knowledge

import (
	"context"
	"log"
	"strings"
)

const defaultThreshold float32 = 0.70

// retriever implements Retriever by dispatching to the appropriate search strategy.
type retriever struct {
	embedder  Embedder
	tokenizer *Tokenizer
	store     Store
}

func (r *retriever) Search(ctx context.Context, query string, mode SearchMode, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if topK <= 0 {
		topK = 5
	}

	switch mode {
	case SearchKeyword:
		return r.searchKeyword(ctx, query, topK)
	case SearchHybrid:
		return r.searchHybrid(ctx, query, topK)
	default:
		return r.searchSemantic(ctx, query, topK)
	}
}

func (r *retriever) searchSemantic(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	vector, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, ErrEmbedFailed
	}
	return r.store.SearchDense(ctx, vector, topK, defaultThreshold)
}

func (r *retriever) searchHybrid(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	dense, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		// Fallback to keyword-only if embedding fails
		log.Printf("knowledge_base: embedding failed, falling back to keyword search: %v", err)
		return r.searchKeyword(ctx, query, topK)
	}

	sparse := r.tokenizer.VectorizeQuery(query)
	if len(sparse.Indices) == 0 {
		// Fallback to semantic-only if BM25 produces empty vector
		return r.store.SearchDense(ctx, dense, topK, defaultThreshold)
	}

	return r.store.SearchHybrid(ctx, dense, sparse, topK)
}

func (r *retriever) searchKeyword(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	sparse := r.tokenizer.VectorizeQuery(query)
	if len(sparse.Indices) == 0 {
		return nil, ErrNoResults
	}
	return r.store.SearchSparse(ctx, sparse, topK)
}
