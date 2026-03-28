package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/cases"
)

type mockCasesService struct {
	createCase      *cases.Case
	createErr       error
	getCase         *cases.Case
	getErr          error
	listCases       []cases.Case
	listErr         error
	changeStatusErr error
}

func (m *mockCasesService) Create(ctx context.Context, body cases.CaseCreate, principalID string) (*cases.Case, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createCase, nil
}

func (m *mockCasesService) Get(ctx context.Context, id string, principal *auth.Principal) (*cases.Case, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getCase, nil
}

func (m *mockCasesService) List(ctx context.Context, filters cases.ListFilters, principal *auth.Principal) ([]cases.Case, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listCases, nil
}

func (m *mockCasesService) Update(ctx context.Context, id string, body cases.CaseUpdate, principal *auth.Principal) (*cases.Case, error) {
	return m.getCase, nil
}

func (m *mockCasesService) Delete(ctx context.Context, id string, principal *auth.Principal) error {
	return nil
}

func (m *mockCasesService) ChangeStatus(ctx context.Context, id string, req cases.StatusChangeRequest, principal *auth.Principal) (*cases.Case, error) {
	if m.changeStatusErr != nil {
		return nil, m.changeStatusErr
	}
	return m.getCase, nil
}

func (m *mockCasesService) Import(ctx context.Context, mode cases.ImportMode, items []cases.CaseImportItem, principal *auth.Principal) (cases.ImportResult, error) {
	return cases.ImportResult{Imported: len(items)}, nil
}

func (m *mockCasesService) ListCategories(ctx context.Context) ([]string, error) { return nil, nil }

func (m *mockCasesService) Export(ctx context.Context, category, status string, principal *auth.Principal) ([]cases.Case, error) {
	return m.listCases, nil
}

func TestHandler_List_Returns200AndArray(t *testing.T) {
	svc := &mockCasesService{listCases: []cases.Case{}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	var list []cases.Case
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if list == nil {
		t.Error("response must be array (not null)")
	}
}

func TestHandler_Create_Returns201AndCase(t *testing.T) {
	created := &cases.Case{
		ID:        "id-1",
		Category:  "billing",
		Title:     "Test",
		Status:    cases.StatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	svc := &mockCasesService{createCase: created}
	h := &Handler{Service: svc}
	body := []byte(`{"category":"billing","title":"Test","response_template":"R"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	var c cases.Case
	if err := json.NewDecoder(rec.Body).Decode(&c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.ID != "id-1" || c.Status != cases.StatusDraft {
		t.Errorf("case id=%s status=%s", c.ID, c.Status)
	}
}

func TestHandler_Create_ValidationError_Returns422(t *testing.T) {
	svc := &mockCasesService{createErr: cases.ErrValidation}
	h := &Handler{Service: svc}
	body := []byte(`{"category":"billing","title":"Test","response_template":"R"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d", rec.Code)
	}
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != "validation_error" {
		t.Errorf("code %s", env.Error.Code)
	}
}

func TestHandler_Get_NotFound_Returns404(t *testing.T) {
	svc := &mockCasesService{getErr: cases.ErrNotFound}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/550e8400-e29b-41d4-a716-446655440000", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestHandler_Get_DraftForbidden_Returns403(t *testing.T) {
	svc := &mockCasesService{getErr: cases.ErrForbidden}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/550e8400-e29b-41d4-a716-446655440000", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestHandler_ChangeStatus_InvalidTransition_Returns422(t *testing.T) {
	svc := &mockCasesService{changeStatusErr: cases.ErrInvalidStatus}
	h := &Handler{Service: svc}
	body := []byte(`{"status":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/550e8400-e29b-41d4-a716-446655440000/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d", rec.Code)
	}
}

// Contract: error response matches OpenAPI ErrorEnvelope.
func TestHandler_Contract_ErrorEnvelope(t *testing.T) {
	svc := &mockCasesService{getErr: cases.ErrNotFound}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/550e8400-e29b-41d4-a716-446655440000", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var env struct {
		Error struct {
			Code    string      `json:"code"`
			Message string      `json:"message"`
			Details interface{} `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != "not_found" {
		t.Errorf("code %s", env.Error.Code)
	}
}

// Contract: Case response has required OpenAPI fields (id, status, created_at, updated_at, etc.).
func TestHandler_Contract_CaseResponseShape(t *testing.T) {
	created := &cases.Case{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Category:  "billing",
		Title:     "Test",
		Status:    cases.StatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	svc := &mockCasesService{createCase: created}
	h := &Handler{Service: svc}
	body := []byte(`{"category":"billing","title":"Test","response_template":"R"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "tid", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var c struct {
		ID        string   `json:"id"`
		Category  string   `json:"category"`
		Title     string   `json:"title"`
		Status    string   `json:"status"`
		CreatedAt string   `json:"created_at"`
		UpdatedAt string   `json:"updated_at"`
		Questions []string `json:"questions"`
		Keywords  []string `json:"keywords"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.ID == "" || c.Status == "" || c.CreatedAt == "" || c.UpdatedAt == "" {
		t.Error("OpenAPI Case: id, status, created_at, updated_at required")
	}
}
