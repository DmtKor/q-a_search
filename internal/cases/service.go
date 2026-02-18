package cases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/cases/fts"
)

// Service implements cases business logic: CRUD, status transitions, embedding lifecycle, import/export.
type Service struct {
	Repo   CaseRepository
	Embed  EmbeddingProvider
}

// Create creates a new case as draft with created_by and search_tsv.
func (s *Service) Create(ctx context.Context, body CaseCreate, principalID string) (*Case, error) {
	if body.Category == "" || body.Title == "" || body.ResponseTemplate == "" {
		return nil, ErrValidation
	}
	c := &Case{
		Category:         body.Category,
		Title:            body.Title,
		Questions:        body.Questions,
		Keywords:         body.Keywords,
		ResponseTemplate: body.ResponseTemplate,
		Status:           StatusDraft,
		CreatedBy:        &principalID,
	}
	if c.Questions == nil {
		c.Questions = []string{}
	}
	if c.Keywords == nil {
		c.Keywords = []string{}
	}
	tsvInput := fts.BuildSearchTSVInput(c.Title, c.Keywords, c.Questions)
	if err := s.Repo.Create(ctx, c, tsvInput); err != nil {
		return nil, err
	}
	return c, nil
}

// Get returns a case by ID; enforces draft-only-for-creator.
func (s *Service) Get(ctx context.Context, id string, principal *auth.Principal) (*Case, error) {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status == StatusDraft && !auth.IsOwner(principal, ptrStr(c.CreatedBy)) {
		return nil, ErrForbidden
	}
	return c, nil
}

// List returns cases with filters; draft only if created_by == principal.
func (s *Service) List(ctx context.Context, filters ListFilters, principal *auth.Principal) ([]Case, error) {
	principalID := ""
	if principal != nil {
		principalID = principal.TokenID
	}
	return s.Repo.List(ctx, filters, principalID)
}

// Update updates a case; cannot change status; recalculates search_tsv; if approved and content changed, refresh embedding.
func (s *Service) Update(ctx context.Context, id string, body CaseUpdate, principal *auth.Principal) (*Case, error) {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status == StatusDraft && !auth.IsOwner(principal, ptrStr(c.CreatedBy)) {
		return nil, ErrForbidden
	}
	// Merge body into current for FTS and repo
	title := c.Title
	keywords := c.Keywords
	questions := c.Questions
	if body.Title != nil {
		title = *body.Title
	}
	if len(body.Keywords) != 0 {
		keywords = body.Keywords
	}
	if len(body.Questions) != 0 {
		questions = body.Questions
	}
	tsvInput := fts.BuildSearchTSVInput(title, keywords, questions)
	updated, err := s.Repo.Update(ctx, id, &body, tsvInput, principal.TokenID)
	if err != nil {
		return nil, err
	}
	// If case is approved and content changed, re-embed and upsert
	if updated.Status == StatusApproved && s.Embed != nil {
		text := embeddingText(keywords, questions)
		vec, err := s.Embed.EmbedQuery(ctx, text)
		if err != nil {
			return nil, err
		}
		if err := s.Repo.UpsertEmbedding(ctx, id, vec); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// Delete soft-deletes (archived) and removes embedding; draft only for creator.
func (s *Service) Delete(ctx context.Context, id string, principal *auth.Principal) error {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.Status == StatusDraft && !auth.IsOwner(principal, ptrStr(c.CreatedBy)) {
		return ErrForbidden
	}
	return s.Repo.SoftDelete(ctx, id)
}

// ChangeStatus applies allowed transitions; embedding lifecycle: approved → upsert, leave approved → delete.
func (s *Service) ChangeStatus(ctx context.Context, id string, req StatusChangeRequest, principal *auth.Principal) (*Case, error) {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from := c.Status
	to := req.Status

	// Idempotent: already approved and request approved → 200
	if from == StatusApproved && to == StatusApproved {
		return c, nil
	}

	// Allowed transitions
	switch {
	case from == StatusDraft && to == StatusPendingReview:
		if !auth.IsOwner(principal, ptrStr(c.CreatedBy)) {
			return nil, ErrForbidden
		}
	case from == StatusPendingReview && (to == StatusApproved || to == StatusDraft):
		// any staff
	case from == StatusApproved && to == StatusArchived:
		// any staff
	default:
		return nil, ErrInvalidStatus
	}

	updated, err := s.Repo.SetStatus(ctx, id, to, req.Comment, principal.TokenID)
	if err != nil {
		return nil, err
	}

	// Embedding lifecycle
	if from == StatusApproved && to != StatusApproved {
		_ = s.Repo.DeleteEmbeddingByCaseID(ctx, id)
	}
	if from != StatusApproved && to == StatusApproved {
		if s.Embed != nil {
			text := embeddingText(updated.Keywords, updated.Questions)
			vec, err := s.Embed.EmbedQuery(ctx, text)
			if err != nil {
				return nil, err
			}
			if err := s.Repo.UpsertEmbedding(ctx, id, vec); err != nil {
				return nil, err
			}
		}
	}

	return updated, nil
}

// Import runs import by mode: replace (delete all + insert), merge (by id), draft (all new draft).
func (s *Service) Import(ctx context.Context, mode ImportMode, items []CaseImportItem, principal *auth.Principal) (ImportResult, error) {
	principalID := principal.TokenID
	var imported, updated int
	var errs []string

	switch mode {
	case ImportModeReplace:
		s.Repo.DeleteAll(ctx)
		imported, errs = s.Repo.ReplaceAll(ctx, items, principalID)
	case ImportModeMerge:
		imported, updated, errs = s.Repo.ImportMerge(ctx, items, principalID)
	case ImportModeDraft:
		imported, errs = s.Repo.ImportDraft(ctx, items, principalID)
	default:
		return ImportResult{}, errors.Join(ErrValidation, fmt.Errorf("invalid mode %q", mode))
	}
	return ImportResult{Imported: imported, Updated: updated, Errors: errs}, nil
}

// Export returns cases by filters (category, status). Drafts are included only when created_by == principal.
func (s *Service) Export(ctx context.Context, category, status string, principal *auth.Principal) ([]Case, error) {
	principalID := ""
	if principal != nil {
		principalID = principal.TokenID
	}
	return s.Repo.List(ctx, ListFilters{Category: category, Status: status}, principalID)
}

func embeddingText(keywords, questions []string) string {
	return strings.TrimSpace(strings.Join(keywords, " ") + " " + strings.Join(questions, " "))
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
