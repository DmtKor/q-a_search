package tickets

import "context"

// TicketsRepository is the persistence interface for tickets.
type TicketsRepository interface {
	Create(ctx context.Context, t *Ticket) error
	GetByID(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context, filters ListFilters) ([]Ticket, error)
	Update(ctx context.Context, id string, u *TicketUpdate) (*Ticket, error)
	SetConvertedToCaseID(ctx context.Context, ticketID, caseID string) error
}
