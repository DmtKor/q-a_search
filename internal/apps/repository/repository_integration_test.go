// Integration tests for apps repository (Postgres).
// Run with: go test -tags=integration ./internal/apps/repository/...
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

	"github.com/yourusername/project/internal/apps"
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
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	return pool
}

func TestIntegration_Create_List_Get_Update(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	a := &apps.App{
		Name:     "MyApp",
		Settings: map[string]interface{}{"search": map[string]interface{}{"default_top_k": 10}},
	}
	err := repo.Create(ctx, a)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.ID == "" {
		t.Fatal("create did not set ID")
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len %d", len(list))
	}
	if list[0].Name != a.Name {
		t.Errorf("list name %s", list[0].Name)
	}

	got, err := repo.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != a.Name {
		t.Errorf("get name %s", got.Name)
	}

	upd := apps.AppUpdate{Name: strPtr("UpdatedApp")}
	updated, err := repo.Update(ctx, a.ID, &upd)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "UpdatedApp" {
		t.Errorf("updated name %s", updated.Name)
	}
}

func TestIntegration_DuplicateName_Conflict(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	a1 := &apps.App{Name: "UniqueName", Settings: map[string]interface{}{}}
	if err := repo.Create(ctx, a1); err != nil {
		t.Fatalf("first create: %v", err)
	}
	a2 := &apps.App{Name: "UniqueName", Settings: map[string]interface{}{}}
	err := repo.Create(ctx, a2)
	if err == nil {
		t.Fatal("expected conflict on duplicate name")
	}
	if err != apps.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestIntegration_Settings_Get_Update_Export_Import_RoundTrip(t *testing.T) {
	connStr, cleanup := startPostgres(t)
	defer cleanup()
	runMigration(t, connStr)
	ctx := context.Background()
	pool := newPool(t, ctx, connStr)
	defer pool.Close()
	repo := NewPool(pool)

	a := &apps.App{Name: "SettingsApp", Settings: map[string]interface{}{}}
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update settings
	settings := map[string]interface{}{
		"search": map[string]interface{}{
			"default_top_k":         25,
			"confidence_threshold":  0.85,
		},
		"extra": "allowed",
	}
	if err := repo.UpdateSettings(ctx, a.ID, settings); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	// Export (GetSettings)
	exported, err := repo.GetSettings(ctx, a.ID)
	if err != nil {
		t.Fatalf("export/get settings: %v", err)
	}
	if exported["extra"] != "allowed" {
		t.Errorf("exported extra: %v", exported["extra"])
	}
	searchVal, ok := exported["search"].(map[string]interface{})
	if !ok {
		t.Fatalf("exported search not map: %T", exported["search"])
	}
	if topK, ok := searchVal["default_top_k"]; !ok {
		t.Error("default_top_k missing")
	} else if f, ok := toFloat(topK); !ok || int(f) != 25 {
		t.Errorf("default_top_k: %v", topK)
	}
	if th, ok := searchVal["confidence_threshold"]; !ok {
		t.Error("confidence_threshold missing")
	} else if f, ok := toFloat(th); !ok || f != 0.85 {
		t.Errorf("confidence_threshold: %v", th)
	}

	// Import (replace entirely) - use exported as import body
	if err := repo.UpdateSettings(ctx, a.ID, exported); err != nil {
		t.Fatalf("import/update settings: %v", err)
	}
	afterImport, err := repo.GetSettings(ctx, a.ID)
	if err != nil {
		t.Fatalf("get after import: %v", err)
	}
	if afterImport["extra"] != "allowed" {
		t.Errorf("after import extra: %v", afterImport["extra"])
	}
	searchAfter := afterImport["search"].(map[string]interface{})
	if searchAfter["default_top_k"] != float64(25) && searchAfter["default_top_k"] != 25 {
		t.Errorf("after import default_top_k: %v", searchAfter["default_top_k"])
	}
}

func strPtr(s string) *string { return &s }

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}
