// Integration tests: real Postgres, migrations, middleware chain, request_metrics.
// Run with: go test -tags=integration ./internal/http/middleware/metrics/...
// Requires Docker.

//go:build integration

package metrics

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	pkgmetrics "github.com/yourusername/project/internal/metrics"
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

func runMigration(t *testing.T, connStr string) {
	t.Helper()
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	migPath := filepath.Join("..", "..", "..", "..", "db", "migrations", "0001_init.sql")
	if _, err := os.Stat(migPath); err != nil {
		migPath = filepath.Join("db", "migrations", "0001_init.sql")
	}
	body, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(body)); err != nil {
		t.Fatalf("run migration: %v", err)
	}
}

func newPool(t *testing.T, ctx context.Context, connStr string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	return pool
}

// testTokenStore implements auth.TokenStore for integration tests.
type testTokenStore struct {
	db *sql.DB
}

func (s *testTokenStore) GetByTokenHash(ctx context.Context, hash string) (*auth.TokenRow, error) {
	var id, tokenHash, tokenType string
	var role, appID sql.NullString
	var isActive bool
	var expiresAt, lastUsedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, token_hash, token_type, role, app_id, is_active, expires_at, last_used_at FROM auth_tokens WHERE token_hash = $1`,
		hash,
	).Scan(&id, &tokenHash, &tokenType, &role, &appID, &isActive, &expiresAt, &lastUsedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row := &auth.TokenRow{ID: id, TokenHash: tokenHash, TokenType: tokenType, IsActive: isActive}
	if role.Valid {
		row.Role = role.String
	}
	if appID.Valid {
		row.AppID = &appID.String
	}
	if expiresAt.Valid {
		row.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		row.LastUsedAt = &lastUsedAt.Time
	}
	return row, nil
}

func (s *testTokenStore) UpdateLastUsedAt(ctx context.Context, tokenID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), tokenID)
	return err
}

func TestIntegration_401_RecordedWithNullTokenIdAppId(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()

	store := pkgmetrics.NewStore(pool)
	tokenStore := &testTokenStore{db: db}
	secret := []byte("test-secret")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := Metrics(store)(authmw.Authenticate(tokenStore, secret)(EnrichPrincipal(inner)))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/v1/cases", nil)
	// No Authorization header -> Auth returns 401
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("response: got %d, want 401", rec.Code)
	}

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM request_metrics WHERE endpoint = $1 AND status_code = 401`, "/api/v1/cases").Scan(&count)
	if err != nil {
		t.Fatalf("query request_metrics: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row in request_metrics for 401, got %d", count)
	}

	var tokenID, appID sql.NullString
	err = pool.QueryRow(ctx,
		`SELECT token_id, app_id FROM request_metrics WHERE endpoint = $1 AND status_code = 401`,
		"/api/v1/cases",
	).Scan(&tokenID, &appID)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}
	if tokenID.Valid || appID.Valid {
		t.Errorf("for 401 request: token_id and app_id must be NULL, got token_id=%v app_id=%v", tokenID, appID)
	}
}

func TestIntegration_ProtectedEndpoint_RecordedWithTokenIdAppId(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()

	// Insert app and app token
	appID := "b1000000-0000-0000-0000-000000000001"
	tokenID := "a0000000-0000-0000-0000-000000000001"
	_, err = db.Exec(`INSERT INTO apps (id, name) VALUES ($1, 'Test App')`, appID)
	if err != nil {
		t.Fatalf("insert app: %v", err)
	}
	secret := []byte("test-secret")
	rawToken := "integration-app-token"
	hash := auth.HashToken(secret, rawToken)
	_, err = db.Exec(
		`INSERT INTO auth_tokens (id, token_hash, token_type, app_id, is_active) VALUES ($1, $2, 'app', $3, true)`,
		tokenID, hash, appID,
	)
	if err != nil {
		t.Fatalf("insert token: %v", err)
	}

	store := pkgmetrics.NewStore(pool)
	tokenStore := &testTokenStore{db: db}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := Metrics(store)(authmw.Authenticate(tokenStore, secret)(EnrichPrincipal(inner)))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/v1/search", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("response: got %d, want 200", rec.Code)
	}

	var gotTokenID, gotAppID sql.NullString
	err = pool.QueryRow(ctx,
		`SELECT token_id, app_id FROM request_metrics WHERE endpoint = $1 AND status_code = 200 ORDER BY created_at DESC LIMIT 1`,
		"/api/v1/search",
	).Scan(&gotTokenID, &gotAppID)
	if err != nil {
		t.Fatalf("query request_metrics: %v", err)
	}
	if !gotTokenID.Valid || gotTokenID.String != tokenID {
		t.Errorf("token_id: got %v, want %s", gotTokenID, tokenID)
	}
	if !gotAppID.Valid || gotAppID.String != appID {
		t.Errorf("app_id: got %v, want %s", gotAppID, appID)
	}
}
