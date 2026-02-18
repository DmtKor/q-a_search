package tickets

import "context"

// CreateDraftFromTicketRequest is the input for creating a draft case from a ticket.
// Used by CaseCreator; filled from ticket + request body + defaults (see spec).
type CreateDraftFromTicketRequest struct {
	Query            string // from ticket.query
	Title            string
	Category         string
	ResponseTemplate string
	CreatedBy        string // principal.TokenID
}

// CaseCreator creates a draft case from a ticket (convert-to-case).
// Defined in module 04; implementation is injected in Glue (e.g. from module 03).
type CaseCreator interface {
	CreateDraftFromTicket(ctx context.Context, req CreateDraftFromTicketRequest) (caseID string, err error)
}
