package search

import (
	"context"

	"github.com/yourusername/project/internal/auth"
)

// EmbeddingProvider produces a vector for the query.
type EmbeddingProvider interface {
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

// SearchRepository returns approved-case candidates (vector + FTS).
type SearchRepository interface {
	SearchApproved(ctx context.Context, params SearchParams) ([]Candidate, error)
}

// TemplateRenderer renders a template with user context.
type TemplateRenderer interface {
	Render(ctx context.Context, template string, userContext map[string]interface{}) (string, error)
}

// LowConfidenceTicketData is passed when creating a ticket for low-confidence search.
type LowConfidenceTicketData struct {
	Query      string
	Category   string
	Confidence float64
}

// TicketsWriter creates tickets (e.g. for low-confidence results).
type TicketsWriter interface {
	CreateLowConfidenceTicket(ctx context.Context, data LowConfidenceTicketData) (ticketID, ticketURL string, err error)
}

// AppSettingsReader provides search settings for the current principal.
type AppSettingsReader interface {
	GetEffectiveSearchSettings(ctx context.Context, principal *auth.Principal) (threshold float64, defaultTopK int, err error)
}
