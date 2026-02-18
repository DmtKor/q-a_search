package metrics

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implements Writer using PostgreSQL request_metrics table.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a Writer that inserts into request_metrics.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Write inserts one row into request_metrics. created_at is set by DB.
func (s *Store) Write(ctx context.Context, record *Record) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO request_metrics (endpoint, status_code, response_time_ms, token_id, app_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		record.Endpoint,
		record.StatusCode,
		record.ResponseTimeMs,
		record.TokenID,
		record.AppID,
	)
	return err
}
