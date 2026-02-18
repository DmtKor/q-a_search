package tickets

import (
	"context"
	"errors"
	"testing"
)

func TestService_Create_ValidatesQuery(t *testing.T) {
	svc := &Service{Repo: &mockRepo{}}
	_, err := svc.Create(context.Background(), TicketCreate{Query: ""})
	if err == nil {
		t.Fatal("expected validation error for empty query")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
	_, err = svc.Create(context.Background(), TicketCreate{Query: "   "})
	if err == nil {
		t.Fatal("expected validation error for whitespace-only query")
	}
}

func TestService_Create_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := &Service{Repo: repo}
	ticket, err := svc.Create(context.Background(), TicketCreate{
		Query:      "How to reset password?",
		Category:   "support",
		Confidence: ptrFloat64(0.5),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ticket.Query != "How to reset password?" {
		t.Errorf("query: got %q", ticket.Query)
	}
	if ticket.Status != StatusOpen {
		t.Errorf("status: got %q", ticket.Status)
	}
	if repo.createCalled != 1 {
		t.Errorf("expected Repo.Create called once, got %d", repo.createCalled)
	}
}

func TestService_Update_ValidatesStatus(t *testing.T) {
	repo := &mockRepo{getTicket: &Ticket{ID: "id1", Status: StatusOpen}}
	svc := &Service{Repo: repo}
	invalid := "invalid_status"
	_, err := svc.Update(context.Background(), "id1", TicketUpdate{Status: &invalid})
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_Update_ResolutionNotesAndAssignedTo(t *testing.T) {
	notes := "Fixed by closing duplicate"
	assigned := "staff-1"
	repo := &mockRepo{
		getTicket: &Ticket{ID: "id1", Status: StatusOpen},
		updateTicket: &Ticket{
			ID: "id1", Status: StatusResolved,
			ResolutionNotes: &notes, AssignedTo: &assigned,
		},
	}
	svc := &Service{Repo: repo}
	updated, err := svc.Update(context.Background(), "id1", TicketUpdate{
		Status:          strPtr(StatusResolved),
		ResolutionNotes: &notes,
		AssignedTo:      &assigned,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ResolutionNotes == nil || *updated.ResolutionNotes != notes {
		t.Errorf("resolution_notes: got %v", updated.ResolutionNotes)
	}
	if updated.AssignedTo == nil || *updated.AssignedTo != assigned {
		t.Errorf("assigned_to: got %v", updated.AssignedTo)
	}
}

func TestService_ConvertToCase_ConflictWhenAlreadyConverted(t *testing.T) {
	caseID := "case-1"
	repo := &mockRepo{getTicket: &Ticket{ID: "t1", Query: "q", ConvertedToCaseID: &caseID}}
	svc := &Service{Repo: repo, CaseCreator: &mockCaseCreator{}}
	_, err := svc.ConvertToCase(context.Background(), "t1", ConvertToCaseRequest{}, "principal-1")
	if err == nil {
		t.Fatal("expected conflict when already converted")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

type mockRepo struct {
	createCalled int
	getTicket    *Ticket
	updateTicket *Ticket
	listTickets  []Ticket
}

func (m *mockRepo) Create(ctx context.Context, t *Ticket) error {
	m.createCalled++
	t.ID = "mock-id"
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*Ticket, error) {
	if m.getTicket != nil {
		return m.getTicket, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepo) List(ctx context.Context, filters ListFilters) ([]Ticket, error) {
	return m.listTickets, nil
}

func (m *mockRepo) Update(ctx context.Context, id string, u *TicketUpdate) (*Ticket, error) {
	if m.updateTicket != nil {
		return m.updateTicket, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepo) SetConvertedToCaseID(ctx context.Context, ticketID, caseID string) error {
	return nil
}

type mockCaseCreator struct {
	caseID string
	err    error
}

func (m *mockCaseCreator) CreateDraftFromTicket(ctx context.Context, req CreateDraftFromTicketRequest) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.caseID != "" {
		return m.caseID, nil
	}
	return "created-case-id", nil
}

func strPtr(s string) *string { return &s }
func ptrFloat64(f float64) *float64 { return &f }
