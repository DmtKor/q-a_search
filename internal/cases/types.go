package cases

import "time"

// Case is the API/DB model (OpenAPI Case).
type Case struct {
	ID               string     `json:"id"`
	Category         string     `json:"category"`
	Title            string     `json:"title"`
	Questions        []string   `json:"questions"`
	Keywords         []string   `json:"keywords"`
	ResponseTemplate string     `json:"response_template"`
	Status           string     `json:"status"`
	CreatedBy        *string    `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedBy        *string    `json:"updated_by,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ApprovedBy       *string    `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

// CaseCreate is the create body (OpenAPI CaseCreate).
type CaseCreate struct {
	Category         string   `json:"category"`
	Title            string   `json:"title"`
	Questions        []string `json:"questions,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	ResponseTemplate string   `json:"response_template"`
}

// CaseUpdate is the update body (OpenAPI CaseUpdate); all fields optional.
type CaseUpdate struct {
	Category         *string  `json:"category,omitempty"`
	Title            *string  `json:"title,omitempty"`
	Questions        []string `json:"questions,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	ResponseTemplate *string  `json:"response_template,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}

// StatusChangeRequest is the body for POST /cases/{id}/status (OpenAPI StatusChangeRequest).
type StatusChangeRequest struct {
	Status  string `json:"status"`
	Comment string `json:"comment,omitempty"`
}

// ListFilters for GET /cases (status, category, mine).
type ListFilters struct {
	Status   string
	Category string
	Mine     bool
}

// ImportMode is merge | draft | replace.
type ImportMode string

const (
	ImportModeMerge  ImportMode = "merge"
	ImportModeDraft  ImportMode = "draft"
	ImportModeReplace ImportMode = "replace"
)

// CaseImportItem is one item in import body; CaseCreate + optional ID for merge.
type CaseImportItem struct {
	ID               string   `json:"id,omitempty"`
	Category         string   `json:"category"`
	Title            string   `json:"title"`
	Questions        []string `json:"questions,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	ResponseTemplate string   `json:"response_template"`
}

// ImportResult is the response of import (OpenAPI ImportResult).
type ImportResult struct {
	Imported int      `json:"imported"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors,omitempty"`
}

// Allowed status values.
const (
	StatusDraft         = "draft"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusArchived      = "archived"
)
