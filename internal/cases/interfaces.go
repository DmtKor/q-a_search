package cases

import (
	"context"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/search"
)

// EmbeddingProvider is the same as search.EmbeddingProvider (injected by Glue).
// Cases uses it only for pending_review → approved to compute embedding from keywords+questions.
type EmbeddingProvider = search.EmbeddingProvider

// CaseService is the business logic interface for cases (implemented by Service).
type CaseService interface {
	Create(ctx context.Context, body CaseCreate, principalID string) (*Case, error)
	Get(ctx context.Context, id string, principal *auth.Principal) (*Case, error)
	List(ctx context.Context, filters ListFilters, principal *auth.Principal) ([]Case, error)
	ListCategories(ctx context.Context) ([]string, error)
	Update(ctx context.Context, id string, body CaseUpdate, principal *auth.Principal) (*Case, error)
	Delete(ctx context.Context, id string, principal *auth.Principal) error
	ChangeStatus(ctx context.Context, id string, req StatusChangeRequest, principal *auth.Principal) (*Case, error)
	Import(ctx context.Context, mode ImportMode, items []CaseImportItem, principal *auth.Principal) (ImportResult, error)
	Export(ctx context.Context, category, status string, principal *auth.Principal) ([]Case, error)
}

// CaseRepository is the persistence interface for cases and case_embeddings.
type CaseRepository interface {
	Create(ctx context.Context, c *Case, searchTSVInput string) error
	GetByID(ctx context.Context, id string) (*Case, error)
	List(ctx context.Context, filters ListFilters, principalID string) ([]Case, error)
	ListCategories(ctx context.Context) ([]string, error)
	Update(ctx context.Context, id string, u *CaseUpdate, searchTSVInput string, updatedBy string) (*Case, error)
	SetStatus(ctx context.Context, id string, status string, comment string, principalID string) (*Case, error)
	SoftDelete(ctx context.Context, id string) error
	UpsertEmbedding(ctx context.Context, caseID string, embedding []float32) error
	DeleteEmbeddingByCaseID(ctx context.Context, caseID string) error
	ReplaceAll(ctx context.Context, items []CaseImportItem, createdBy string) (imported int, errs []string)
	ImportMerge(ctx context.Context, items []CaseImportItem, createdBy string) (imported, updated int, errs []string)
	ImportDraft(ctx context.Context, items []CaseImportItem, createdBy string) (imported int, errs []string)
	DeleteAll(ctx context.Context) error
}
