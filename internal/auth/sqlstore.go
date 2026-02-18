package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SQLTokenStore implements TokenStore using PostgreSQL (pgxpool).
// Use NewSQLTokenStore to create; same pool can be used for other repos.
type SQLTokenStore struct {
	pool *pgxpool.Pool
}

// NewSQLTokenStore returns a TokenStore that reads/writes auth_tokens via the given pool.
func NewSQLTokenStore(pool *pgxpool.Pool) *SQLTokenStore {
	return &SQLTokenStore{pool: pool}
}

// Ensure SQLTokenStore implements TokenStore.
var _ TokenStore = (*SQLTokenStore)(nil)

// GetByTokenHash returns the token row for the given hash, or nil if not found.
func (s *SQLTokenStore) GetByTokenHash(ctx context.Context, hash string) (*TokenRow, error) {
	var id, tokenHash, tokenType string
	var role, appID *string
	var isActive bool
	var expiresAt, lastUsedAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT id, token_hash, token_type, role, app_id, is_active, expires_at, last_used_at FROM auth_tokens WHERE token_hash = $1`,
		hash,
	).Scan(&id, &tokenHash, &tokenType, &role, &appID, &isActive, &expiresAt, &lastUsedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &TokenRow{
		ID:         id,
		TokenHash:  tokenHash,
		TokenType:  tokenType,
		Role:       ptrToEmpty(role),
		AppID:      appID,
		IsActive:   isActive,
		ExpiresAt:  expiresAt,
		LastUsedAt: lastUsedAt,
	}, nil
}

// UpdateLastUsedAt sets last_used_at to now for the given token ID.
func (s *SQLTokenStore) UpdateLastUsedAt(ctx context.Context, tokenID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth_tokens SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), tokenID)
	return err
}

func ptrToEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
