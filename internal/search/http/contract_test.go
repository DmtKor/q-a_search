package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/search"
)

// Contract test: response shape matches OpenAPI SearchResponse (chunks[], optional ticket).
func TestHandler_Contract_ResponseMatchesOpenAPI(t *testing.T) {
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
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Decode and validate shape: SearchResponse { chunks[], ticket? }
	var resp struct {
		Chunks []struct {
			CaseID     string  `json:"case_id"`
			Title      string  `json:"title"`
			Text       string  `json:"text"`
			Confidence float64 `json:"confidence"`
		} `json:"chunks"`
		Ticket *struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"ticket"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Required: chunks (array)
	if resp.Chunks == nil {
		t.Error("OpenAPI: chunks must be present (array)")
	}
	// Chunk required fields
	for i, c := range resp.Chunks {
		if c.CaseID == "" {
			t.Errorf("chunk[%d]: case_id required", i)
		}
		if c.Title == "" {
			t.Errorf("chunk[%d]: title required", i)
		}
		// text and confidence required by schema
		if c.Confidence < 0 || c.Confidence > 1 {
			t.Errorf("chunk[%d]: confidence must be 0..1 per OpenAPI", i)
		}
	}
	// ticket optional; when present must have id and url
	if resp.Ticket != nil {
		if resp.Ticket.ID == "" || resp.Ticket.URL == "" {
			t.Error("ticket: when present, id and url required")
		}
	}
}

// Contract test: error response matches OpenAPI ErrorEnvelope.
func TestHandler_Contract_ErrorMatchesOpenAPI(t *testing.T) {
	h := &Handler{Service: &search.Service{AppSettings: &mockSettings{}}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "app"}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}

	var env struct {
		Error struct {
			Code    string      `json:"code"`
			Message string      `json:"message"`
			Details interface{} `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if env.Error.Code != "validation_error" {
		t.Errorf("expected code validation_error, got %s", env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Error("error message required")
	}
}
