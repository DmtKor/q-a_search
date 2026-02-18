package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/project/internal/auth"
)

type mockTokenStore struct {
	byHash map[string]*auth.TokenRow
	lastUsed []string
}

func (m *mockTokenStore) GetByTokenHash(ctx context.Context, hash string) (*auth.TokenRow, error) {
	return m.byHash[hash], nil
}

func (m *mockTokenStore) UpdateLastUsedAt(ctx context.Context, tokenID string) error {
	m.lastUsed = append(m.lastUsed, tokenID)
	return nil
}

func TestAuthenticate_NoToken_401(t *testing.T) {
	store := &mockTokenStore{byHash: map[string]*auth.TokenRow{}}
	secret := []byte("secret")
	handler := Authenticate(store, secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
	if rec.Body.String() != `{"error":{"code":"unauthorized","message":"Missing or invalid authorization token","details":null}}`+"\n" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestAuthenticate_InvalidToken_401(t *testing.T) {
	store := &mockTokenStore{byHash: map[string]*auth.TokenRow{}}
	secret := []byte("secret")
	handler := Authenticate(store, secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer unknown-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
}

func TestAuthenticate_ValidToken_SuccessAndUpdateLastUsed(t *testing.T) {
	secret := []byte("secret")
	rawToken := "my-token"
	hash := auth.HashToken(secret, rawToken)
	row := &auth.TokenRow{
		ID:        "token-uuid-1",
		TokenType: "staff",
		IsActive:  true,
	}
	store := &mockTokenStore{byHash: map[string]*auth.TokenRow{hash: row}}
	handler := Authenticate(store, secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := auth.PrincipalFromContext(r.Context())
		if p == nil {
			t.Error("principal should be in context")
			return
		}
		if p.TokenID != "token-uuid-1" || p.TokenType != "staff" {
			t.Errorf("principal = %+v", p)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
	if len(store.lastUsed) != 1 || store.lastUsed[0] != "token-uuid-1" {
		t.Errorf("UpdateLastUsedAt should have been called with token-uuid-1, got %v", store.lastUsed)
	}
}

func TestAuthenticate_ExpiredToken_401(t *testing.T) {
	secret := []byte("secret")
	rawToken := "my-token"
	hash := auth.HashToken(secret, rawToken)
	past := time.Now().Add(-time.Hour)
	row := &auth.TokenRow{
		ID:        "token-uuid-1",
		TokenType: "staff",
		IsActive:  true,
		ExpiresAt: &past,
	}
	store := &mockTokenStore{byHash: map[string]*auth.TokenRow{hash: row}}
	handler := Authenticate(store, secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
}

func TestAuthenticate_DisabledToken_401(t *testing.T) {
	secret := []byte("secret")
	rawToken := "my-token"
	hash := auth.HashToken(secret, rawToken)
	row := &auth.TokenRow{
		ID:        "token-uuid-1",
		TokenType: "staff",
		IsActive:  false,
	}
	store := &mockTokenStore{byHash: map[string]*auth.TokenRow{hash: row}}
	handler := Authenticate(store, secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
}

func TestRequireStaff_NoPrincipal_403(t *testing.T) {
	handler := RequireStaff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("got status %d, want 403", rec.Code)
	}
}

func TestRequireStaff_AppToken_403(t *testing.T) {
	handler := RequireStaff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "x", TokenType: "app"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("got status %d, want 403", rec.Code)
	}
}

func TestRequireStaff_StaffToken_200(t *testing.T) {
	handler := RequireStaff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "x", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
}

func TestRequireAppOrStaff_App_200(t *testing.T) {
	handler := RequireAppOrStaff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "x", TokenType: "app"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
}

func TestRequireAppOrStaff_Staff_200(t *testing.T) {
	handler := RequireAppOrStaff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenID: "x", TokenType: "staff"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
}
