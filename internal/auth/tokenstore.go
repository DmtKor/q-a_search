package auth

import (
	"context"
	"time"
)

// TokenRow represents a row from auth_tokens used for principal and validation.
type TokenRow struct {
	ID         string
	TokenHash  string
	TokenType  string
	Role       string
	AppID      *string
	IsActive   bool
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
}

// TokenStore is the interface for token lookup and last-used update.
// Implementation (e.g. using *sql.DB) is provided by Glue; this package only uses the interface.
type TokenStore interface {
	GetByTokenHash(ctx context.Context, hash string) (*TokenRow, error)
	UpdateLastUsedAt(ctx context.Context, tokenID string) error
}
