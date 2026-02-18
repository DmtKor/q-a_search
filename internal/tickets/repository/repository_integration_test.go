// Integration tests for tickets repository (Postgres).
// Run with: go test -tags=integration ./internal/tickets/repository/...
// Requires Docker.

//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/yourusername/project/internal/tickets"
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

func TestIntegration_TicketCRUD(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()
	repo := NewPool(pool)

	ticket := &tickets.Ticket{
		Query:      "How to reset password?",
		Status:     tickets.StatusOpen,
		Category:   catPtr("support"),
		Confidence: confPtr(0.6),
	}
	if err := repo.Create(ctx, ticket); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ticket.ID == "" {
		t.Fatal("Create should set ID")
	}

	got, err := repo.GetByID(ctx, ticket.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Query != ticket.Query || got.Status != tickets.StatusOpen {
		t.Errorf("GetByID: got query=%q status=%q", got.Query, got.Status)
	}

	list, err := repo.List(ctx, tickets.ListFilters{Category: "support"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List category=support: expected 1, got %d", len(list))
	}

	notes := "Resolved via email"
	updated, err := repo.Update(ctx, ticket.ID, &tickets.TicketUpdate{
		Status:          catPtr(tickets.StatusResolved),
		ResolutionNotes: &notes,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != tickets.StatusResolved || updated.ResolutionNotes == nil || *updated.ResolutionNotes != notes {
		t.Errorf("Update: got status=%q notes=%v", updated.Status, updated.ResolutionNotes)
	}
}

func TestIntegration_ConvertToCase_SetsConvertedToCaseID(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()
	repo := NewPool(pool)

	// Create case manually (simulating CaseCreator in Glue)
	var caseID string
	err = pool.QueryRow(ctx, `INSERT INTO cases (id, category, title, questions, response_template, status, created_by)
		VALUES (gen_random_uuid(), 'general', 'From ticket', '["query"]'::jsonb, '{{.Query}}', 'draft', 'principal-1')
		RETURNING id`).Scan(&caseID)
	if err != nil {
		t.Fatalf("insert case: %v", err)
	}

	ticket := &tickets.Ticket{Query: "User question", Status: tickets.StatusOpen}
	if err := repo.Create(ctx, ticket); err != nil {
		t.Fatalf("Create ticket: %v", err)
	}

	if err := repo.SetConvertedToCaseID(ctx, ticket.ID, caseID); err != nil {
		t.Fatalf("SetConvertedToCaseID: %v", err)
	}
	got, err := repo.GetByID(ctx, ticket.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ConvertedToCaseID == nil || *got.ConvertedToCaseID != caseID {
		t.Errorf("converted_to_case_id: got %v", got.ConvertedToCaseID)
	}
}

func catPtr(s string) *string { return &s }
func confPtr(f float64) *float64 { return &f }
