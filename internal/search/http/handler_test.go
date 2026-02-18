package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	"github.com/yourusername/project/internal/search"
)

func TestHandler_MissingQuery_422(t *testing.T) {
	svc := &search.Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{},
		Renderer:    &mockRenderer{},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{},
	}
	h := &Handler{Service: svc}

	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "app"}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
	var env authmw.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "validation_error" {
		t.Errorf("expected validation_error, got %s", env.Error.Code)
	}
}

func TestHandler_InvalidTopK_422(t *testing.T) {
	svc := &search.Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{},
		Renderer:    &mockRenderer{},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{},
	}
	h := &Handler{Service: svc}

	body := []byte(`{"query":"x","top_k":0}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "app"}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
}

func TestHandler_Success_200(t *testing.T) {
	svc := &search.Service{
		Embedding:   &mockEmbedding{vec: []float32{0.1}},
		Repo:        &mockRepo{candidates: []search.Candidate{{CaseID: "c1", Title: "T", ResponseTemplate: "R", CosineSimilarity: 0.9}}},
		Renderer:    &mockRenderer{out: "Rendered"},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{},
	}
	h := &Handler{Service: svc}

	body := []byte(`{"query":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "app"}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp search.SearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(resp.Chunks))
	}
	if resp.Chunks[0].CaseID != "c1" || resp.Chunks[0].Confidence != 0.9 || resp.Chunks[0].Text != "Rendered" {
		t.Errorf("chunk: %+v", resp.Chunks[0])
	}
	if resp.Ticket != nil {
		t.Error("expected no ticket")
	}
}

func TestHandler_NoPrincipal_403(t *testing.T) {
	h := &Handler{Service: &search.Service{AppSettings: &mockSettings{}}}
	body := []byte(`{"query":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// no principal in context

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// Mocks for handler tests (same package so we need concrete types in this file or import from search_test)
type mockEmbedding struct{ vec []float32 }

func (m *mockEmbedding) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if m.vec != nil {
		return m.vec, nil
	}
	return []float32{0.1}, nil
}

type mockRepo struct{ candidates []search.Candidate }

func (m *mockRepo) SearchApproved(ctx context.Context, params search.SearchParams) ([]search.Candidate, error) {
	return m.candidates, nil
}

type mockRenderer struct{ out string }

func (m *mockRenderer) Render(ctx context.Context, template string, userContext map[string]interface{}) (string, error) {
	if m.out != "" {
		return m.out, nil
	}
	return template, nil
}

type mockTickets struct{}

func (m *mockTickets) CreateLowConfidenceTicket(ctx context.Context, data search.LowConfidenceTicketData) (string, string, error) {
	return "tid", "/tickets/tid", nil
}

type mockSettings struct{}

func (m *mockSettings) GetEffectiveSearchSettings(ctx context.Context, principal *auth.Principal) (float64, int, error) {
	return 0.7, 10, nil
}
