package tickets

import "time"

// Ticket is the API/DB model (OpenAPI Ticket).
type Ticket struct {
	ID                 string     `json:"id"`
	Query              string     `json:"query"`
	Category           *string    `json:"category,omitempty"`
	Confidence         *float64   `json:"confidence,omitempty"`
	Status             string     `json:"status"`
	AssignedTo         *string    `json:"assigned_to,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes    *string    `json:"resolution_notes,omitempty"`
	ConvertedToCaseID  *string    `json:"converted_to_case_id,omitempty"`
}

// TicketCreate is the create body (OpenAPI TicketCreate); query required.
type TicketCreate struct {
	Query      string   `json:"query"`
	Category   string   `json:"category,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
}

// TicketUpdate is the update body (OpenAPI TicketUpdate); all fields optional.
type TicketUpdate struct {
	Status          *string `json:"status,omitempty"`
	AssignedTo      *string `json:"assigned_to,omitempty"`
	ResolutionNotes *string `json:"resolution_notes,omitempty"`
}

// ListFilters for GET /tickets (status, category).
type ListFilters struct {
	Status   string
	Category string
}

// ConvertToCaseRequest is the body for POST /tickets/{id}/convert-to-case (OpenAPI).
type ConvertToCaseRequest struct {
	Title            string `json:"title,omitempty"`
	Category         string `json:"category,omitempty"`
	ResponseTemplate string `json:"response_template,omitempty"`
}

// ConvertToCaseResponse is the response (OpenAPI).
type ConvertToCaseResponse struct {
	CaseID string `json:"case_id"`
	URL    string `json:"url"`
}

// Ticket status values (OpenAPI enum).
const (
	StatusOpen        = "open"
	StatusInProgress  = "in_progress"
	StatusResolved    = "resolved"
	StatusClosed      = "closed"
)

// ValidStatuses for workflow validation.
var ValidStatuses = map[string]bool{
	StatusOpen: true, StatusInProgress: true, StatusResolved: true, StatusClosed: true,
}

const (
	DefaultTitleMaxLen     = 200
	DefaultCategory        = "general"
	DefaultResponseTpl     = "{{.Query}}"
	DefaultTitleFromTicket = "From ticket"
)
