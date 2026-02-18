package embedding

import (
	"context"

	"github.com/yourusername/project/internal/search"
)

// StubDimension matches case_embeddings.embedding vector(1536).
const StubDimension = 1536

// Stub is an EmbeddingProvider that returns a fixed zero vector (or constant vector).
// Use for e2e, dev, and demo when no real embedding API is configured.
type Stub struct{}

// NewStub returns a stub embedding provider.
func NewStub() *Stub {
	return &Stub{}
}

// EmbedQuery implements search.EmbeddingProvider. Returns a zero vector of StubDimension.
func (s *Stub) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	_ = ctx
	_ = query
	vec := make([]float32, StubDimension)
	return vec, nil
}

// Ensure Stub implements search.EmbeddingProvider.
var _ search.EmbeddingProvider = (*Stub)(nil)
