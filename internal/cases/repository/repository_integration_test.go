// Integration tests for cases repository (Postgres + pgvector + FTS).
// Run with: go test -tags=integration ./internal/cases/repository/...
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
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/yourusername/project/internal/cases"
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

func newPool(t *testing.T, ctx context.Context, connStr string) *pgxpool.Pool {
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

func TestIntegration_Create_List_Get_Update_SoftDelete(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	c := &cases.Case{
		Category:         "billing",
		Title:            "Cancel subscription",
		Questions:        []string{"how to cancel"},
		Keywords:        []string{"cancel", "subscription"},
		ResponseTemplate: "Go to Settings.",
		Status:           cases.StatusDraft,
		CreatedBy:        toStrPtr("staff-1"),
	}
	tsv := "Cancel subscription cancel subscription how to cancel"
	err := repo.Create(ctx, c, tsv)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == "" {
		t.Fatal("create did not set ID")
	}

	list, err := repo.List(ctx, cases.ListFilters{}, "staff-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len %d", len(list))
	}
	if list[0].Title != c.Title {
		t.Errorf("list title %s", list[0].Title)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Category != c.Category {
		t.Errorf("get category %s", got.Category)
	}

	upd := cases.CaseUpdate{Title: toStrPtr("Updated title")}
	updated, err := repo.Update(ctx, c.ID, &upd, "Updated title cancel subscription how to cancel", "staff-1")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("updated title %s", updated.Title)
	}

	err = repo.SoftDelete(ctx, c.ID)
	if err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	_, err = repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("get after delete (archived): %v", err)
	}
	// Case still exists with status archived
	got2, _ := repo.GetByID(ctx, c.ID)
	if got2.Status != cases.StatusArchived {
		t.Errorf("status after delete %s", got2.Status)
	}
}

func TestIntegration_Status_Approved_Embedding_Lifecycle(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	c := &cases.Case{
		Category:         "support",
		Title:            "Password reset",
		Questions:        []string{"reset password"},
		Keywords:        []string{"password"},
		ResponseTemplate: "Use Forgot password.",
		Status:           cases.StatusDraft,
		CreatedBy:        toStrPtr("staff-1"),
	}
	tsv := "Password reset password reset password"
	if err := repo.Create(ctx, c, tsv); err != nil {
		t.Fatalf("create: %v", err)
	}

	// draft -> pending_review
	_, err := repo.SetStatus(ctx, c.ID, cases.StatusPendingReview, "", "staff-1")
	if err != nil {
		t.Fatalf("set pending_review: %v", err)
	}
	// No embedding for pending_review
	conn, _ := pool.Acquire(ctx)
	var n int
	err = conn.Conn().QueryRow(ctx, `SELECT COUNT(*) FROM case_embeddings WHERE case_id = $1`, c.ID).Scan(&n)
	conn.Release()
	if err != nil || n != 0 {
		t.Errorf("expected 0 embeddings for pending_review, got %d err %v", n, err)
	}

	// pending_review -> approved; then we upsert embedding from service. Here we only test repo.
	_, err = repo.SetStatus(ctx, c.ID, cases.StatusApproved, "", "staff-1")
	if err != nil {
		t.Fatalf("set approved: %v", err)
	}
	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = 0.1
	}
	if err := repo.UpsertEmbedding(ctx, c.ID, vec); err != nil {
		t.Fatalf("upsert embedding: %v", err)
	}
	var count int
	conn2, _ := pool.Acquire(ctx)
	err = conn2.Conn().QueryRow(ctx, `SELECT COUNT(*) FROM case_embeddings WHERE case_id = $1`, c.ID).Scan(&count)
	conn2.Release()
	if err != nil || count != 1 {
		t.Fatalf("expected 1 embedding, got %d err %v", count, err)
	}

	// approved -> archived: delete embedding
	if err := repo.DeleteEmbeddingByCaseID(ctx, c.ID); err != nil {
		t.Fatalf("delete embedding: %v", err)
	}
	_, err = repo.SetStatus(ctx, c.ID, cases.StatusArchived, "", "staff-1")
	if err != nil {
		t.Fatalf("set archived: %v", err)
	}
	conn3, _ := pool.Acquire(ctx)
	err = conn3.Conn().QueryRow(ctx, `SELECT COUNT(*) FROM case_embeddings WHERE case_id = $1`, c.ID).Scan(&count)
	conn3.Release()
	if err != nil || count != 0 {
		t.Errorf("expected 0 embeddings after archived, got %d err %v", count, err)
	}
}

func TestIntegration_SearchTSV_Filled(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	c := &cases.Case{
		Category:         "billing",
		Title:            "Invoice",
		Questions:        []string{"get invoice"},
		Keywords:        []string{"invoice"},
		ResponseTemplate: "Download from Billing.",
		Status:           cases.StatusDraft,
		CreatedBy:        toStrPtr("staff-1"),
	}
	tsv := "Invoice invoice get invoice"
	if err := repo.Create(ctx, c, tsv); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = got
	conn, _ := pool.Acquire(ctx)
	var hasTsv bool
	err = conn.Conn().QueryRow(ctx, `SELECT search_tsv IS NOT NULL FROM cases WHERE id = $1`, c.ID).Scan(&hasTsv)
	conn.Release()
	if err != nil {
		t.Fatalf("read search_tsv: %v", err)
	}
	if !hasTsv {
		t.Error("search_tsv should be set")
	}
}

func TestIntegration_ReplaceAll_DeletesAndInserts(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	// Create one case
	c := &cases.Case{
		Category:         "x",
		Title:            "First",
		ResponseTemplate: "T",
		Status:           cases.StatusDraft,
		CreatedBy:        toStrPtr("s1"),
	}
	if err := repo.Create(ctx, c, "First"); err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeID := c.ID

	// Replace with two new items
	items := []cases.CaseImportItem{
		{Category: "a", Title: "A", ResponseTemplate: "R1"},
		{Category: "b", Title: "B", ResponseTemplate: "R2"},
	}
	imported, errs := repo.ReplaceAll(ctx, items, "s1")
	if len(errs) != 0 {
		t.Fatalf("replace errs: %v", errs)
	}
	if imported != 2 {
		t.Errorf("imported %d", imported)
	}

	// Old case should be gone
	_, err := repo.GetByID(ctx, beforeID)
	if err == nil {
		t.Error("old case should not exist")
	}

	list, _ := repo.List(ctx, cases.ListFilters{}, "s1")
	if len(list) != 2 {
		t.Errorf("list len %d", len(list))
	}
}

func toStrPtr(s string) *string { return &s }
