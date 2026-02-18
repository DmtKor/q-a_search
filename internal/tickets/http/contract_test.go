package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/tickets"
)

// Contract test: Ticket response shape matches OpenAPI (Ticket schema).
func TestHandler_Contract_TicketResponseShape(t *testing.T) {
	svc := &tickets.Service{Repo: &fixedTicketRepo{}}
	h := &Handler{Service: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "p1", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var ticket struct {
		ID                string   `json:"id"`
		Query             string   `json:"query"`
		Category          *string  `json:"category"`
		Confidence        *float64 `json:"confidence"`
		Status            string   `json:"status"`
		AssignedTo        *string  `json:"assigned_to"`
		CreatedAt         string   `json:"created_at"`
		UpdatedAt         string   `json:"updated_at"`
		ResolvedAt        *string  `json:"resolved_at"`
		ResolutionNotes   *string  `json:"resolution_notes"`
		ConvertedToCaseID *string  `json:"converted_to_case_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&ticket); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ticket.ID == "" || ticket.Query == "" || ticket.Status == "" {
		t.Error("OpenAPI Ticket: id, query, status required")
	}
	validStatus := map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true}
	if !validStatus[ticket.Status] {
		t.Errorf("status must be one of open|in_progress|resolved|closed, got %q", ticket.Status)
	}
}

// Contract test: TicketListResponse is array of Ticket.
func TestHandler_Contract_ListResponseShape(t *testing.T) {
	svc := &tickets.Service{Repo: &fixedTicketRepo{list: []tickets.Ticket{}}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	// OpenAPI TicketListResponse = array of Ticket
	if list == nil {
		t.Error("chunks must be present (array)")
	}
}

// Contract test: ConvertToCaseResponse has case_id and url.
func TestHandler_Contract_ConvertToCaseResponseShape(t *testing.T) {
	ticketID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	svc := &tickets.Service{
		Repo:        &fixedTicketRepo{getTicket: &tickets.Ticket{ID: ticketID, Query: "q"}},
		CaseCreator: &fixedCaseCreator{caseID: "case-uuid-1"},
	}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/"+ticketID+"/convert-to-case", bytes.NewReader([]byte("{}")))
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "p1", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		CaseID string `json:"case_id"`
		URL    string `json:"url"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CaseID == "" || resp.URL == "" {
		t.Error("ConvertToCaseResponse: case_id and url required per OpenAPI")
	}
}

// Contract test: Error response matches OpenAPI ErrorEnvelope.
func TestHandler_Contract_ErrorEnvelope(t *testing.T) {
	svc := &tickets.Service{Repo: &fixedTicketRepo{}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/00000000-0000-0000-0000-000000000000", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing ticket, got %d", rec.Code)
	}
	var env struct {
		Error struct {
			Code    string      `json:"code"`
			Message string      `json:"message"`
			Details interface{} `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if env.Error.Code != "not_found" || env.Error.Message == "" {
		t.Errorf("ErrorEnvelope: code=not_found, message required; got code=%q", env.Error.Code)
	}
}

type fixedTicketRepo struct {
	getTicket *tickets.Ticket
	list      []tickets.Ticket
}

func (f *fixedTicketRepo) Create(ctx context.Context, t *tickets.Ticket) error { return nil }
func (f *fixedTicketRepo) GetByID(ctx context.Context, id string) (*tickets.Ticket, error) {
	if f.getTicket != nil && f.getTicket.ID == id {
		return f.getTicket, nil
	}
	if id == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		return &tickets.Ticket{
			ID:     id,
			Query:  "test query",
			Status: tickets.StatusOpen,
		}, nil
	}
	return nil, tickets.ErrNotFound
}
func (f *fixedTicketRepo) List(ctx context.Context, filters tickets.ListFilters) ([]tickets.Ticket, error) {
	return f.list, nil
}
func (f *fixedTicketRepo) Update(ctx context.Context, id string, u *tickets.TicketUpdate) (*tickets.Ticket, error) {
	return nil, tickets.ErrNotFound
}
func (f *fixedTicketRepo) SetConvertedToCaseID(ctx context.Context, ticketID, caseID string) error {
	return nil
}

type fixedCaseCreator struct {
	caseID string
}

func (f *fixedCaseCreator) CreateDraftFromTicket(ctx context.Context, req tickets.CreateDraftFromTicketRequest) (string, error) {
	if f.caseID != "" {
		return f.caseID, nil
	}
	return "case-uuid-1", nil
}
