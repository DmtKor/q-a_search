package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/project/internal/apps"
	"github.com/yourusername/project/internal/auth"
)

// fixedRepo returns one app for get/list and creates for create.
type fixedAppsRepo struct {
	app       *apps.App
	list      []apps.App
	createErr error
	getErr    error
	settings  map[string]interface{}
}

func (f *fixedAppsRepo) Create(ctx context.Context, a *apps.App) error {
	if f.createErr != nil {
		return f.createErr
	}
	a.ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	return nil
}
func (f *fixedAppsRepo) GetByID(ctx context.Context, id string) (*apps.App, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.app != nil && f.app.ID == id {
		return f.app, nil
	}
	return nil, apps.ErrNotFound
}
func (f *fixedAppsRepo) List(ctx context.Context) ([]apps.App, error) {
	return f.list, nil
}
func (f *fixedAppsRepo) Update(ctx context.Context, id string, u *apps.AppUpdate) (*apps.App, error) {
	return nil, apps.ErrNotFound
}
func (f *fixedAppsRepo) GetSettings(ctx context.Context, appID string) (map[string]interface{}, error) {
	if f.settings != nil {
		return f.settings, nil
	}
	return nil, apps.ErrNotFound
}
func (f *fixedAppsRepo) UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error {
	return nil
}

// Contract test: App response shape (OpenAPI App schema).
func TestHandler_Contract_AppResponseShape(t *testing.T) {
	appID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	svc := &apps.Service{Repo: &fixedAppsRepo{app: &apps.App{
		ID:   appID,
		Name: "TestApp",
		Settings: map[string]interface{}{
			"search": map[string]interface{}{"default_top_k": 10},
		},
	}}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+appID, nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var app struct {
		ID        string                 `json:"id"`
		Name      string                 `json:"name"`
		Settings  map[string]interface{} `json:"settings"`
		CreatedAt string                 `json:"created_at"`
		UpdatedAt string                 `json:"updated_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&app); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if app.ID == "" || app.Name == "" {
		t.Error("OpenAPI App: id, name required")
	}
}

// Contract test: List returns array of App.
func TestHandler_Contract_ListResponseShape(t *testing.T) {
	svc := &apps.Service{Repo: &fixedAppsRepo{list: []apps.App{}}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list == nil {
		t.Error("list must be present (array)")
	}
}

// Contract test: Error envelope (404).
func TestHandler_Contract_ErrorEnvelope(t *testing.T) {
	svc := &apps.Service{Repo: &fixedAppsRepo{getErr: apps.ErrNotFound}}
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/00000000-0000-0000-0000-000000000000", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var env struct {
		Error struct {
			Code    string      `json:"code"`
			Message string      `json:"message"`
			Details interface{} `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if env.Error.Code != "not_found" || env.Error.Message == "" {
		t.Errorf("ErrorEnvelope: code=not_found; got code=%q", env.Error.Code)
	}
}

// Contract test: Settings GET returns object (AppSettings).
func TestHandler_Contract_SettingsResponseShape(t *testing.T) {
	appID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	svc := &apps.Service{Repo: &fixedAppsRepo{settings: map[string]interface{}{
		"search": map[string]interface{}{"default_top_k": 10, "confidence_threshold": 0.7},
	}}}
	// GetSettings uses GetSettings by appID - but our fixedAppsRepo GetSettings returns f.settings for any id. We need to fix repo to return settings for this app. Actually our fixed repo returns settings when f.settings != nil, and GetByID is used for Get. So GetSettings is called with id from path. So we need a repo that returns settings for that appID. Our fixedAppsRepo.GetSettings returns f.settings regardless of appID. So we need to make the handler get id from path and call GetSettings(ctx, id). So when we request GET /api/v1/apps/aaa.../settings, the id is aaa..., and GetSettings(ctx, "aaa...") is called. Our mock returns f.settings. So we're good.
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+appID+"/settings", nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var settings map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings["search"] == nil {
		t.Error("AppSettings may contain search object")
	}
}

// Contract test: 422 on invalid settings (default_top_k out of range).
func TestHandler_Contract_ValidationError(t *testing.T) {
	appID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	svc := &apps.Service{Repo: &fixedAppsRepo{app: &apps.App{ID: appID, Name: "A"}}}
	body := map[string]interface{}{"search": map[string]interface{}{"default_top_k": 100}}
	bodyBytes, _ := json.Marshal(body)
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/apps/"+appID+"/settings", bytes.NewReader(bodyBytes))
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{TokenType: "staff"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid default_top_k, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != "validation_error" {
		t.Errorf("expected code=validation_error, got %q", env.Error.Code)
	}
}
