// Integration tests for auth against real PostgreSQL.
// Run with: go test -tags=integration ./internal/auth/...
// Requires Docker.

//go:build integration

package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startPostgres(t *testing.T) (connStr string, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "pgvector/pgvector:pg16",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres: %v", err)
	}
	connStr, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("connection string: %v", err)
	}
	return connStr, func() { _ = testcontainers.TerminateContainer(container) }
}

func runMigrationPool(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	migPath := filepath.Join("..", "..", "db", "migrations", "0001_init.sql")
	if _, err := os.Stat(migPath); err != nil {
		migPath = filepath.Join("db", "migrations", "0001_init.sql")
	}
	body, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("run migration: %v", err)
	}
}

func TestIntegration_TokenActive_SuccessAndLastUsedUpdated(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	runMigrationPool(t, ctx, pool)
	store := NewSQLTokenStore(pool)

	secret := []byte("integration-secret")
	rawToken := "integration-raw-token"
	hash := HashToken(secret, rawToken)
	_, err = pool.Exec(ctx, `INSERT INTO auth_tokens (id, token_hash, token_type, is_active) VALUES ($1, $2, 'staff', true)`,
		"a0000000-0000-0000-0000-000000000001", hash)
	if err != nil {
		t.Fatalf("insert token: %v", err)
	}

	row, err := store.GetByTokenHash(ctx, hash)
	if err != nil || row == nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if !row.IsActive {
		t.Error("token should be active")
	}
	_ = store.UpdateLastUsedAt(ctx, row.ID)

	var lastUsed *time.Time
	err = pool.QueryRow(ctx, `SELECT last_used_at FROM auth_tokens WHERE id = $1`, row.ID).Scan(&lastUsed)
	if err != nil {
		t.Fatalf("select last_used_at: %v", err)
	}
	if lastUsed == nil {
		t.Error("last_used_at should have been set")
	}
}

func TestIntegration_TokenExpired_NotFound(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	runMigrationPool(t, ctx, pool)
	store := NewSQLTokenStore(pool)

	secret := []byte("integration-secret")
	rawToken := "expired-token"
	hash := HashToken(secret, rawToken)
	expired := time.Now().Add(-time.Hour)
	_, err = pool.Exec(ctx, `INSERT INTO auth_tokens (id, token_hash, token_type, is_active, expires_at) VALUES ($1, $2, 'staff', true, $3)`,
		"a0000000-0000-0000-0000-000000000002", hash, expired)
	if err != nil {
		t.Fatalf("insert token: %v", err)
	}

	row, err := store.GetByTokenHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if row == nil {
		t.Fatal("row should exist")
	}
	if row.ExpiresAt == nil || !row.ExpiresAt.Before(time.Now()) {
		t.Error("token should be expired for middleware to return 401")
	}
}

func TestIntegration_TokenInactive_Rejected(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	runMigrationPool(t, ctx, pool)
	store := NewSQLTokenStore(pool)

	secret := []byte("integration-secret")
	rawToken := "inactive-token"
	hash := HashToken(secret, rawToken)
	_, err = pool.Exec(ctx, `INSERT INTO auth_tokens (id, token_hash, token_type, is_active) VALUES ($1, $2, 'staff', false)`,
		"a0000000-0000-0000-0000-000000000003", hash)
	if err != nil {
		t.Fatalf("insert token: %v", err)
	}

	row, err := store.GetByTokenHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if row == nil {
		t.Fatal("row should exist")
	}
	if row.IsActive {
		t.Error("token should be inactive for middleware to return 401")
	}
}
