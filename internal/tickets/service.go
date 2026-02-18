package tickets

import (
	"context"
	"strings"

	"github.com/yourusername/project/internal/search"
)

// Service implements tickets business logic: CRUD, status workflow, convert-to-case.
type Service struct {
	Repo           TicketsRepository
	CaseCreator    CaseCreator
	TicketsBaseURL string // e.g. "/api/v1/tickets" for ticket URL in search response
}

// List returns tickets with optional filters.
func (s *Service) List(ctx context.Context, filters ListFilters) ([]Ticket, error) {
	return s.Repo.List(ctx, filters)
}

// Get returns a ticket by ID.
func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	return s.Repo.GetByID(ctx, id)
}

// Create creates a new ticket (manual staff creation).
func (s *Service) Create(ctx context.Context, body TicketCreate) (*Ticket, error) {
	if strings.TrimSpace(body.Query) == "" {
		return nil, ErrValidation
	}
	t := &Ticket{
		Query:   strings.TrimSpace(body.Query),
		Status:  StatusOpen,
		Confidence: body.Confidence,
	}
	if body.Category != "" {
		t.Category = &body.Category
	}
	if err := s.Repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update updates a ticket; validates status transition (any of open|in_progress|resolved|closed).
func (s *Service) Update(ctx context.Context, id string, body TicketUpdate) (*Ticket, error) {
	if body.Status != nil && !ValidStatuses[*body.Status] {
		return nil, ErrValidation
	}
	return s.Repo.Update(ctx, id, &body)
}

// ConvertToCase creates a draft case from the ticket and sets converted_to_case_id.
func (s *Service) ConvertToCase(ctx context.Context, id string, body ConvertToCaseRequest, principalID string) (*ConvertToCaseResponse, error) {
	t, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.ConvertedToCaseID != nil && *t.ConvertedToCaseID != "" {
		return nil, ErrConflict
	}
	title := body.Title
	if title == "" {
		title = trimToLen(t.Query, DefaultTitleMaxLen)
	}
	if title == "" {
		title = DefaultTitleFromTicket
	}
	category := body.Category
	if category == "" && t.Category != nil {
		category = *t.Category
	}
	if category == "" {
		category = DefaultCategory
	}
	responseTemplate := body.ResponseTemplate
	if responseTemplate == "" {
		responseTemplate = DefaultResponseTpl
	}
	caseID, err := s.CaseCreator.CreateDraftFromTicket(ctx, CreateDraftFromTicketRequest{
		Query:            t.Query,
		Title:            title,
		Category:         category,
		ResponseTemplate: responseTemplate,
		CreatedBy:        principalID,
	})
	if err != nil {
		return nil, err
	}
	if err := s.Repo.SetConvertedToCaseID(ctx, id, caseID); err != nil {
		return nil, err
	}
	return &ConvertToCaseResponse{
		CaseID: caseID,
		URL:    "/api/v1/cases/" + caseID,
	}, nil
}

// CreateLowConfidenceTicket creates a ticket from search low-confidence flow (implements search.TicketsWriter).
func (s *Service) CreateLowConfidenceTicket(ctx context.Context, data search.LowConfidenceTicketData) (ticketID, ticketURL string, err error) {
	t := &Ticket{
		Query:      data.Query,
		Status:     StatusOpen,
		Confidence: &data.Confidence,
	}
	if data.Category != "" {
		t.Category = &data.Category
	}
	if err := s.Repo.Create(ctx, t); err != nil {
		return "", "", err
	}
	base := s.TicketsBaseURL
	if base == "" {
		base = "/api/v1/tickets"
	}
	return t.ID, strings.TrimSuffix(base, "/") + "/" + t.ID, nil
}

func trimToLen(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// Ensure Service implements search.TicketsWriter at compile time.
var _ search.TicketsWriter = (*Service)(nil)
