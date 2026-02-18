package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/project/internal/auth"
	pkgmetrics "github.com/yourusername/project/internal/metrics"
)

type mockWriter struct {
	last *pkgmetrics.Record
	err  error
}

func (m *mockWriter) Write(ctx context.Context, rec *pkgmetrics.Record) error {
	m.last = rec
	return m.err
}

func TestMetrics_EndpointIsRawPath(t *testing.T) {
	mw := &mockWriter{}
	handler := Metrics(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/v1/cases/550e8400-e29b-42d4-a716-446655440000", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	// Endpoint is raw path (r.URL.Path), not route template.
	if mw.last.Endpoint != "/api/v1/cases/550e8400-e29b-42d4-a716-446655440000" {
		t.Errorf("endpoint: got %q, want raw path /api/v1/cases/550e8400-e29b-42d4-a716-446655440000", mw.last.Endpoint)
	}
}

func TestMetrics_StatusCodeCaptured(t *testing.T) {
	mw := &mockWriter{}
	handler := Metrics(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "https://example.com/api/v1/cases", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	if mw.last.StatusCode != http.StatusCreated {
		t.Errorf("status_code: got %d, want %d", mw.last.StatusCode, http.StatusCreated)
	}
}

func TestMetrics_StatusCodeDefault200(t *testing.T) {
	mw := &mockWriter{}
	handler := Metrics(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	if mw.last.StatusCode != http.StatusOK {
		t.Errorf("status_code when WriteHeader not called: got %d, want 200", mw.last.StatusCode)
	}
}

func TestMetrics_ResponseTimeMsMeasured(t *testing.T) {
	mw := &mockWriter{}
	handler := Metrics(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	if mw.last.ResponseTimeMs < 10 {
		t.Errorf("response_time_ms: got %d, want at least 10", mw.last.ResponseTimeMs)
	}
}

func TestMetrics_WriteErrorDoesNotBreakResponse(t *testing.T) {
	mw := &mockWriter{err: context.DeadlineExceeded}
	handler := Metrics(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("response code: got %d, want 200 (metrics write error must not break response)", rec.Code)
	}
}

func TestMetrics_EnrichPrincipal_SetsTokenIdAndAppId(t *testing.T) {
	mw := &mockWriter{}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := Metrics(mw)(fakeAuthWithPrincipal(
		&auth.Principal{TokenID: "token-uuid-1234", TokenType: "app", AppID: strPtr("app-uuid-5678")},
	)(EnrichPrincipal(inner)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	if mw.last.TokenID == nil || *mw.last.TokenID != "token-uuid-1234" {
		t.Errorf("token_id: got %v, want token-uuid-1234", mw.last.TokenID)
	}
	if mw.last.AppID == nil || *mw.last.AppID != "app-uuid-5678" {
		t.Errorf("app_id: got %v, want app-uuid-5678", mw.last.AppID)
	}
}

func TestMetrics_WithoutEnrichPrincipal_TokenIdAppIdNil(t *testing.T) {
	mw := &mockWriter{}
	// Chain without EnrichPrincipal: Auth sets principal in context but Metrics never sees it.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := Metrics(mw)(fakeAuthWithPrincipal(
		&auth.Principal{TokenID: "tid", AppID: strPtr("app-id")},
	)(inner))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)
	if mw.last == nil {
		t.Fatal("writer was not called")
	}
	if mw.last.TokenID != nil || mw.last.AppID != nil {
		t.Errorf("without EnrichPrincipal: token_id=%v app_id=%v, want both nil", mw.last.TokenID, mw.last.AppID)
	}
}

// fakeAuthWithPrincipal is a stub that puts principal in context and calls next (simulates Auth + next).
func fakeAuthWithPrincipal(p *auth.Principal) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.WithPrincipal(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
