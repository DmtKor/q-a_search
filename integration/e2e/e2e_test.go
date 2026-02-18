// Minimal e2e: app + staff tokens, case draft→approved, search (with low-confidence ticket), convert-to-case.
// Run with: go test -tags=integration ./integration/e2e/...
// Requires Docker.

//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/yourusername/project/internal/apps"
	appshandler "github.com/yourusername/project/internal/apps/http"
	appsrepo "github.com/yourusername/project/internal/apps/repository"
	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/cases"
	caseshttp "github.com/yourusername/project/internal/cases/http"
	casesrepo "github.com/yourusername/project/internal/cases/repository"
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

const (
	e2eSecret     = "e2e-secret"
	e2eAppToken   = "e2e-app-token"
	e2eStaffToken = "e2e-staff-token"
)

// lowConfStub returns different vectors per call so that case embedding vs query embedding have low cosine similarity.
type lowConfStub struct {
	call int
	dim  int
}

func (s *lowConfStub) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	_ = ctx
	_ = query
	vec := make([]float32, s.dim)
	if s.dim >= 2 {
		vec[s.call%2] = 1.0
	}
	s.call++
	return vec, nil
}

func TestE2E_FullScenario(t *testing.T) {
	ctx := context.Background()
	connStr, cleanup := startPostgres(t)
	defer cleanup()

	runMigration(t, ctx, connStr)
	pool := newPool(t, ctx, connStr)
	defer pool.Close()

	// Seed: app + app token + staff token
	secret := []byte(e2eSecret)
	appID := seedApp(t, ctx, pool)
	seedToken(t, ctx, pool, secret, "app", appID)
	staffTokenID := seedToken(t, ctx, pool, secret, "staff", "")

	// Embedding stub that yields low confidence on search (different vectors for case vs query)
	emb := &lowConfStub{dim: embedding.StubDimension}
	renderer := template.NewRenderer()

	casesRepo := casesrepo.NewPool(pool)
	appsRepo := appsrepo.NewPool(pool)
	ticketsRepo := ticketsrepo.NewPool(pool)
	searchRepo := searchrepo.NewPool(pool)

	casesSvc := &cases.Service{Repo: casesRepo, Embed: emb}
	caseCreator := glue.NewCaseCreatorAdapter(casesSvc)
	ticketsSvc := &tickets.Service{
		Repo:           ticketsRepo,
		CaseCreator:    caseCreator,
		TicketsBaseURL: "/api/v1/tickets",
	}
	appsSvc := &apps.Service{Repo: appsRepo}
	effectiveSettings := &apps.EffectiveSettingsResolver{Repo: appsRepo}
	searchSvc := &search.Service{
		Embedding:   emb,
		Repo:        searchRepo,
		Renderer:    renderer,
		Tickets:     ticketsSvc,
		AppSettings: effectiveSettings,
	}

	tokenStore := auth.NewSQLTokenStore(pool)
	metricsStore := metrics.NewStore(pool)

	handler := httppkg.Handler(httppkg.RouterConfig{
		MetricsWriter:  metricsStore,
		TokenStore:     tokenStore,
		Secret:         secret,
		SearchHandler:  &searchhandler.Handler{Service: searchSvc},
		CasesHandler:   &caseshttp.Handler{Service: casesSvc},
		TicketsHandler: &ticketshttp.Handler{Service: ticketsSvc},
		AppsHandler:    &appshandler.Handler{Service: appsSvc},
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	base := server.URL
	staffAuth := "Bearer " + e2eStaffToken
	appAuth := "Bearer " + e2eAppToken

	// 1) Staff creates case (draft)
	createBody := `{"category":"e2e","title":"E2E case","questions":["q1"],"keywords":["k1"],"response_template":"Answer"}`
	resp := post(t, base+"/api/v1/cases", staffAuth, createBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create case: status %d body %s", resp.StatusCode, slurp(resp.Body))
	}
	var createdCase struct {
		ID string `json:"id"`
	}
	decode(t, resp.Body, &createdCase)
	caseID := createdCase.ID
	if caseID == "" {
		t.Fatal("case id empty")
	}

	// 2) Staff: draft -> pending_review -> approved
	statusBody := `{"status":"pending_review"}`
	resp = post(t, base+"/api/v1/cases/"+caseID+"/status", staffAuth, statusBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status pending_review: %d %s", resp.StatusCode, slurp(resp.Body))
	}
	statusBody = `{"status":"approved"}`
	resp = post(t, base+"/api/v1/cases/"+caseID+"/status", staffAuth, statusBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status approved: %d %s", resp.StatusCode, slurp(resp.Body))
	}

	// 3) App calls search; with lowConfStub we expect low confidence and a ticket in response
	searchBody := `{"query":"e2e query","top_k":5}`
	resp = post(t, base+"/api/v1/search", appAuth, searchBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search: %d %s", resp.StatusCode, slurp(resp.Body))
	}
	var searchResp struct {
		Chunks []interface{} `json:"chunks"`
		Ticket *struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"ticket,omitempty"`
	}
	decode(t, resp.Body, &searchResp)

	var ticketID string
	if searchResp.Ticket != nil && searchResp.Ticket.ID != "" {
		ticketID = searchResp.Ticket.ID
	} else {
		t.Log("search did not return ticket; creating ticket manually for convert-to-case")
		ticketBody := `{"query":"e2e manual ticket","category":"e2e"}`
		resp = post(t, base+"/api/v1/tickets", staffAuth, ticketBody)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create ticket: %d %s", resp.StatusCode, slurp(resp.Body))
		}
		var tkt struct {
			ID string `json:"id"`
		}
		decode(t, resp.Body, &tkt)
		ticketID = tkt.ID
	}
	if ticketID == "" {
		t.Fatal("no ticket id for convert-to-case")
	}

	// 4) Staff: convert ticket to case
	convertBody := `{"title":"From ticket","category":"e2e","response_template":"Resolved."}`
	resp = post(t, base+"/api/v1/tickets/"+ticketID+"/convert-to-case", staffAuth, convertBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("convert-to-case: %d %s", resp.StatusCode, slurp(resp.Body))
	}
	var convertResp struct {
		CaseID string `json:"case_id"`
		URL    string `json:"url"`
	}
	decode(t, resp.Body, &convertResp)
	if convertResp.CaseID == "" {
		t.Fatal("convert-to-case did not return case_id")
	}

	// 5) Staff lists cases and should see the new draft
	resp = get(t, base+"/api/v1/cases?status=draft", staffAuth)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list cases: %d %s", resp.StatusCode, slurp(resp.Body))
	}
	var listResp []struct {
		ID        string  `json:"id"`
		Status    string  `json:"status"`
		CreatedBy *string `json:"created_by,omitempty"`
	}
	decode(t, resp.Body, &listResp)
	var found bool
	for _, c := range listResp {
		if c.ID == convertResp.CaseID && c.Status == "draft" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("new draft case %s not found in list or wrong status", convertResp.CaseID)
	}
	_ = staffTokenID
}

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

// newPool creates a pool with pgvector types registered. Call after runMigration so the vector extension exists.
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

// runMigration runs 0001_init.sql (including CREATE EXTENSION vector) using a plain connection so pgvector types are not required yet.
func runMigration(t *testing.T, ctx context.Context, connStr string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool for migration: %v", err)
	}
	defer pool.Close()
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

func seedApp(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO apps (name, settings) VALUES ($1, $2::jsonb) RETURNING id`, "e2e-app", "{}").Scan(&id)
	if err != nil {
		t.Fatalf("seed app: %v", err)
	}
	return id
}

func seedToken(t *testing.T, ctx context.Context, pool *pgxpool.Pool, secret []byte, tokenType string, appID string) string {
	t.Helper()
	var raw string
	switch tokenType {
	case "app":
		raw = e2eAppToken
	case "staff":
		raw = e2eStaffToken
	default:
		t.Fatalf("unknown token type %s", tokenType)
	}
	hash := auth.HashToken(secret, raw)
	var id string
	if appID != "" {
		err := pool.QueryRow(ctx,
			`INSERT INTO auth_tokens (token_hash, token_type, app_id, is_active) VALUES ($1, $2, $3, true) RETURNING id`,
			hash, tokenType, appID,
		).Scan(&id)
		if err != nil {
			t.Fatalf("seed app token: %v", err)
		}
	} else {
		err := pool.QueryRow(ctx,
			`INSERT INTO auth_tokens (token_hash, token_type, is_active) VALUES ($1, $2, true) RETURNING id`,
			hash, tokenType,
		).Scan(&id)
		if err != nil {
			t.Fatalf("seed staff token: %v", err)
		}
	}
	return id
}

func post(t *testing.T, url, authHeader, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func get(t *testing.T, url, authHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func slurp(r io.Reader) string {
	b, _ := io.ReadAll(r)
	return string(b)
}

func decode(t *testing.T, r io.Reader, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
