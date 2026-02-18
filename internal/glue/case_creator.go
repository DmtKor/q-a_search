package glue

import (
	"context"

	"github.com/yourusername/project/internal/cases"
	"github.com/yourusername/project/internal/tickets"
)

// CaseCreatorAdapter adapts cases.CaseService to tickets.CaseCreator (convert-to-case).
type CaseCreatorAdapter struct {
	Cases cases.CaseService
}

// NewCaseCreatorAdapter returns an adapter that creates draft cases via the cases service.
func NewCaseCreatorAdapter(svc cases.CaseService) *CaseCreatorAdapter {
	return &CaseCreatorAdapter{Cases: svc}
}

// CreateDraftFromTicket implements tickets.CaseCreator.
func (a *CaseCreatorAdapter) CreateDraftFromTicket(ctx context.Context, req tickets.CreateDraftFromTicketRequest) (string, error) {
	body := cases.CaseCreate{
		Category:         req.Category,
		Title:            req.Title,
		Questions:        nil,
		Keywords:         nil,
		ResponseTemplate: req.ResponseTemplate,
	}
	c, err := a.Cases.Create(ctx, body, req.CreatedBy)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// Ensure CaseCreatorAdapter implements tickets.CaseCreator.
var _ tickets.CaseCreator = (*CaseCreatorAdapter)(nil)
