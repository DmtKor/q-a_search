package cases

import (
	"context"
	"errors"
	"testing"

	"github.com/yourusername/project/internal/auth"
)

type mockRepo struct {
	caseByID     map[string]*Case
	createErr    error
	getErr      error
	updateErr   error
	setStatusErr error
	deleteErr   error
	listCases   []Case
	listErr     error
}

func (m *mockRepo) Create(ctx context.Context, c *Case, searchTSVInput string) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.caseByID == nil {
		m.caseByID = make(map[string]*Case)
	}
	cp := *c
	if cp.ID == "" {
		cp.ID = "mock-id-1"
		c.ID = cp.ID
	}
	m.caseByID[cp.ID] = &cp
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*Case, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if c, ok := m.caseByID[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepo) List(ctx context.Context, filters ListFilters, principalID string) ([]Case, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listCases, nil
}

func (m *mockRepo) Update(ctx context.Context, id string, u *CaseUpdate, searchTSVInput string, updatedBy string) (*Case, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	c, ok := m.caseByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *mockRepo) SetStatus(ctx context.Context, id string, status string, comment string, principalID string) (*Case, error) {
	if m.setStatusErr != nil {
		return nil, m.setStatusErr
	}
	c, ok := m.caseByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	cp.Status = status
	return &cp, nil
}

func (m *mockRepo) SoftDelete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.caseByID[id]; !ok {
		return ErrNotFound
	}
	delete(m.caseByID, id)
	return nil
}

func (m *mockRepo) UpsertEmbedding(ctx context.Context, caseID string, embedding []float32) error { return nil }
func (m *mockRepo) DeleteEmbeddingByCaseID(ctx context.Context, caseID string) error             { return nil }
func (m *mockRepo) ReplaceAll(ctx context.Context, items []CaseImportItem, createdBy string) (int, []string) {
	return len(items), nil
}
func (m *mockRepo) ImportMerge(ctx context.Context, items []CaseImportItem, createdBy string) (int, int, []string) {
	return len(items), 0, nil
}
func (m *mockRepo) ImportDraft(ctx context.Context, items []CaseImportItem, createdBy string) (int, []string) {
	return len(items), nil
}
func (m *mockRepo) DeleteAll(ctx context.Context) error { return nil }

func TestService_Get_DraftOnlyCreator(t *testing.T) {
	ctx := context.Background()
	ownerID := "owner-token-id"
	otherID := "other-token-id"
	c := &Case{
		ID:       "case-1",
		Status:   StatusDraft,
		CreatedBy: &ownerID,
	}
	repo := &mockRepo{caseByID: map[string]*Case{"case-1": c}}
	svc := &Service{Repo: repo}

	// Owner can get
	owner := &auth.Principal{TokenID: ownerID, TokenType: "staff"}
	got, err := svc.Get(ctx, "case-1", owner)
	if err != nil {
		t.Fatalf("owner get: %v", err)
	}
	if got.ID != "case-1" {
		t.Errorf("got id %s", got.ID)
	}

	// Other staff cannot get draft
	other := &auth.Principal{TokenID: otherID, TokenType: "staff"}
	_, err = svc.Get(ctx, "case-1", other)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestService_Get_NonDraftAnyStaff(t *testing.T) {
	ctx := context.Background()
	ownerID := "owner-token-id"
	otherID := "other-token-id"
	c := &Case{
		ID:        "case-1",
		Status:    StatusApproved,
		CreatedBy: &ownerID,
	}
	repo := &mockRepo{caseByID: map[string]*Case{"case-1": c}}
	svc := &Service{Repo: repo}

	other := &auth.Principal{TokenID: otherID, TokenType: "staff"}
	got, err := svc.Get(ctx, "case-1", other)
	if err != nil {
		t.Fatalf("staff get approved: %v", err)
	}
	if got.Status != StatusApproved {
		t.Errorf("got status %s", got.Status)
	}
}

func TestService_ChangeStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()
	c := &Case{ID: "case-1", Status: StatusDraft, CreatedBy: ptr("tid")}
	repo := &mockRepo{caseByID: map[string]*Case{"case-1": c}}
	svc := &Service{Repo: repo}
	principal := &auth.Principal{TokenID: "tid", TokenType: "staff"}

	// draft -> approved not allowed
	_, err := svc.ChangeStatus(ctx, "case-1", StatusChangeRequest{Status: StatusApproved}, principal)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_ChangeStatus_IdempotentApprove(t *testing.T) {
	ctx := context.Background()
	c := &Case{ID: "case-1", Status: StatusApproved, CreatedBy: ptr("tid")}
	repo := &mockRepo{caseByID: map[string]*Case{"case-1": c}}
	svc := &Service{Repo: repo}
	principal := &auth.Principal{TokenID: "tid", TokenType: "staff"}

	got, err := svc.ChangeStatus(ctx, "case-1", StatusChangeRequest{Status: StatusApproved}, principal)
	if err != nil {
		t.Fatalf("idempotent approve: %v", err)
	}
	if got.Status != StatusApproved {
		t.Errorf("got status %s", got.Status)
	}
}

func TestService_Create_Validation(t *testing.T) {
	ctx := context.Background()
	svc := &Service{Repo: &mockRepo{}}
	principal := &auth.Principal{TokenID: "tid", TokenType: "staff"}

	_, err := svc.Create(ctx, CaseCreate{Title: "x", ResponseTemplate: "y"}, principal.TokenID)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation (missing category), got %v", err)
	}
}

func ptr(s string) *string { return &s }
