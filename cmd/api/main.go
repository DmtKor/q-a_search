package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/yourusername/project/internal/apps"
	appshandler "github.com/yourusername/project/internal/apps/http"
	appsrepo "github.com/yourusername/project/internal/apps/repository"
	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/cases"
	caseshttp "github.com/yourusername/project/internal/cases/http"
	casesrepo "github.com/yourusername/project/internal/cases/repository"
	"github.com/yourusername/project/internal/config"
	"github.com/yourusername/project/internal/embedding"
	"github.com/yourusername/project/internal/glue"
	httppkg "github.com/yourusername/project/internal/http"
	"github.com/yourusername/project/internal/metrics"
	"github.com/yourusername/project/internal/search"
	searchhandler "github.com/yourusername/project/internal/search/http"
	searchrepo "github.com/yourusername/project/internal/search/repository"
	"github.com/yourusername/project/internal/template"
	"github.com/yourusername/project/internal/tickets"
	ticketshttp "github.com/yourusername/project/internal/tickets/http"
	ticketsrepo "github.com/yourusername/project/internal/tickets/repository"
)

func main() {
	cfg := config.Load()
	if cfg.DSN == "" {
		log.Fatal("DATABASE_URL or POSTGRES_DSN must be set")
	}

	ctx := context.Background()

	// Run migrations first using a pool without pgvector type registration, so CREATE EXTENSION vector runs before any connection uses the vector type.
	migPoolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		log.Fatalf("parse pool config: %v", err)
	}
	migPool, err := pgxpool.NewWithConfig(ctx, migPoolConfig)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	if err := runMigrations(ctx, migPool); err != nil {
		migPool.Close()
		log.Fatalf("migrations: %v", err)
	}
	migPool.Close()

	// Main pool with pgvector types registered (extension already exists after migrations).
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		log.Fatalf("parse pool config: %v", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := checkPgVector(ctx, pool); err != nil {
		log.Fatalf("pgvector check: %v", err)
	}

	// Auth & metrics
	tokenStore := auth.NewSQLTokenStore(pool)
	metricsStore := metrics.NewStore(pool)
	metricsWriter := metricsStore

	// Embedding stub and template renderer
	embeddingProvider := embedding.NewStub()
	renderer := template.NewRenderer(template.WithMaxOutputLen(cfg.TemplateMaxOutputLen))

	// Repos
	casesRepo := casesrepo.NewPool(pool)
	appsRepo := appsrepo.NewPool(pool)
	ticketsRepo := ticketsrepo.NewPool(pool)
	searchRepo := searchrepo.NewPool(pool)

	// Services
	casesSvc := &cases.Service{Repo: casesRepo, Embed: embeddingProvider}
	caseCreator := glue.NewCaseCreatorAdapter(casesSvc)
	ticketsSvc := &tickets.Service{
		Repo:           ticketsRepo,
		CaseCreator:    caseCreator,
		TicketsBaseURL: cfg.TicketsBaseURL,
	}
	appsSvc := &apps.Service{Repo: appsRepo}
	effectiveSettings := &apps.EffectiveSettingsResolver{Repo: appsRepo}
	searchSvc := &search.Service{
		Embedding:   embeddingProvider,
		Repo:        searchRepo,
		Renderer:    renderer,
		Tickets:     ticketsSvc,
		AppSettings: effectiveSettings,
	}

	// Handlers
	searchHandler := &searchhandler.Handler{Service: searchSvc}
	casesHandler := &caseshttp.Handler{Service: casesSvc}
	categoriesHandler := &caseshttp.CategoriesHandler{Service: casesSvc}
	ticketsHandler := &ticketshttp.Handler{Service: ticketsSvc}
	appsHandler := &appshandler.Handler{Service: appsSvc}
	templatePreviewHandler := &template.PreviewHandler{Renderer: renderer}
	templateReadableHandler := &template.ReadableHandler{}

	handler := httppkg.Handler(httppkg.RouterConfig{
		MetricsWriter:          metricsWriter,
		TokenStore:             tokenStore,
		Secret:                 cfg.Secret,
		RequestLogLevel:        cfg.RequestLogLevel,
		TemplatePreviewHandler: templatePreviewHandler,
		TemplateReadableHandler: templateReadableHandler,
		SearchHandler:          searchHandler,
		CasesHandler:           casesHandler,
		CategoriesHandler:      categoriesHandler,
		TicketsHandler:         ticketsHandler,
		AppsHandler:            appsHandler,
	})

	addr := getEnv("LISTEN_ADDR", ":8080")
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migPath := filepath.Join("db", "migrations", "0001_init.sql")
	if _, err := os.Stat(migPath); err != nil {
		migPath = filepath.Join("..", "..", "db", "migrations", "0001_init.sql")
	}
	body, err := os.ReadFile(migPath)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(body))
	return err
}

func checkPgVector(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	err := pool.QueryRow(ctx, `SELECT 1 FROM pg_extension WHERE extname = 'vector'`).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("pgvector extension not found: run CREATE EXTENSION vector")
		}
		return err
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
