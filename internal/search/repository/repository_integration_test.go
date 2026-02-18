// Integration tests for search repository (Postgres + pgvector + FTS).
// Run with: go test -tags=integration ./internal/search/repository/...
// Requires Docker.

//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/yourusername/project/internal/search"
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
	migPath := filepath.Join("..", "..", "..", "db", "migrations", "0001_init.sql")
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

func newPoolWithPgVector(t *testing.T, ctx context.Context, connStr string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	return pool
}

func TestIntegration_SearchApproved_ExcludesDraftPendingArchived(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPoolWithPgVector(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	// Insert cases: one approved, one draft, one pending_review, one archived
	approvedID := "b1000000-0000-0000-0000-000000000001"
	draftID := "b2000000-0000-0000-0000-000000000002"
	pendingID := "b3000000-0000-0000-0000-000000000003"
	archivedID := "b4000000-0000-0000-0000-000000000004"
	for _, row := range []struct {
		id, status string
	}{
		{approvedID, "approved"},
		{draftID, "draft"},
		{pendingID, "pending_review"},
		{archivedID, "archived"},
	} {
		_, err := pool.Exec(ctx, `INSERT INTO cases (id, category, title, questions, response_template, status)
			VALUES ($1, 'cat', 'Title', '[]', 'Template', $2)`, row.id, row.status)
		if err != nil {
			t.Fatalf("insert case: %v", err)
		}
	}
	// Embedding only for approved (as per app invariant)
	vecSlice := make([]float32, 1536)
	for i := range vecSlice {
		vecSlice[i] = 0.01
	}
	vecSlice[0] = 0.5
	vec := pgvector.NewVector(vecSlice)
	_, err := pool.Exec(ctx, `INSERT INTO case_embeddings (case_id, embedding) VALUES ($1, $2)`, approvedID, vec)
	if err != nil {
		t.Fatalf("insert embedding: %v", err)
	}

	params := search.SearchParams{
		QueryVector: vecSlice,
		QueryFTS:    "title",
		Category:    "",
		Limit:       10,
	}
	candidates, err := repo.SearchApproved(ctx, params)
	if err != nil {
		t.Fatalf("SearchApproved: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (only approved), got %d", len(candidates))
	}
	if candidates[0].CaseID != approvedID {
		t.Errorf("expected approved case id %s, got %s", approvedID, candidates[0].CaseID)
	}
}

func TestIntegration_SearchApproved_CategoryFilter(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPoolWithPgVector(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	id1 := "c1000000-0000-0000-0000-000000000001"
	id2 := "c2000000-0000-0000-0000-000000000002"
	_, err := pool.Exec(ctx, `INSERT INTO cases (id, category, title, questions, response_template, status) VALUES ($1, 'sales', 'Sales', '[]', 'T', 'approved')`, id1)
	if err != nil {
		t.Fatalf("insert case 1: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO cases (id, category, title, questions, response_template, status) VALUES ($1, 'support', 'Support', '[]', 'T', 'approved')`, id2)
	if err != nil {
		t.Fatalf("insert case 2: %v", err)
	}
	vecSlice := make([]float32, 1536)
	for i := range vecSlice {
		vecSlice[i] = 0.01
	}
	vec := pgvector.NewVector(vecSlice)
	_, err = pool.Exec(ctx, `INSERT INTO case_embeddings (case_id, embedding) VALUES ($1, $2), ($3, $2)`, id1, vec, id2)
	if err != nil {
		t.Fatalf("insert embeddings: %v", err)
	}

	paramsAll := search.SearchParams{QueryVector: vecSlice, QueryFTS: "x", Category: "", Limit: 10}
	candidatesAll, err := repo.SearchApproved(ctx, paramsAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidatesAll) != 2 {
		t.Fatalf("no category filter: expected 2, got %d", len(candidatesAll))
	}

	paramsCat := search.SearchParams{QueryVector: vecSlice, QueryFTS: "x", Category: "sales", Limit: 10}
	candidatesCat, err := repo.SearchApproved(ctx, paramsCat)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidatesCat) != 1 {
		t.Fatalf("category=sales: expected 1, got %d", len(candidatesCat))
	}
	if candidatesCat[0].Category != "sales" {
		t.Errorf("expected category sales, got %s", candidatesCat[0].Category)
	}
}

func TestIntegration_SearchApproved_FTS(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPoolWithPgVector(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	id := "d1000000-0000-0000-0000-000000000001"
	_, err := pool.Exec(ctx, `INSERT INTO cases (id, category, title, questions, response_template, status, search_tsv)
		VALUES ($1, 'cat', 'Refund policy document', '[]', 'T', 'approved', to_tsvector('english', 'Refund policy document'))`, id)
	if err != nil {
		t.Fatalf("insert case: %v", err)
	}
	vecSlice := make([]float32, 1536)
	for i := range vecSlice {
		vecSlice[i] = 0.01
	}
	vec := pgvector.NewVector(vecSlice)
	_, err = pool.Exec(ctx, `INSERT INTO case_embeddings (case_id, embedding) VALUES ($1, $2)`, id, vec)
	if err != nil {
		t.Fatalf("insert embedding: %v", err)
	}

	params := search.SearchParams{
		QueryVector: vecSlice,
		QueryFTS:    "Refund",
		Category:    "",
		Limit:       10,
	}
	candidates, err := repo.SearchApproved(ctx, params)
	if err != nil {
		t.Fatalf("SearchApproved: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].FTSRank <= 0 {
		t.Error("expected FTS rank > 0 when query matches search_tsv")
	}
}
