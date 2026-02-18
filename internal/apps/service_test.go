package apps

import (
	"context"
	"errors"
	"testing"
)

func TestService_Create_EmptyName_ValidationError(t *testing.T) {
	svc := &Service{Repo: &serviceTestRepo{}}
	_, err := svc.Create(context.Background(), AppCreate{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestService_Create_InvalidSettings_ValidationError(t *testing.T) {
	svc := &Service{Repo: &serviceTestRepo{}}
	_, err := svc.Create(context.Background(), AppCreate{
		Name:     "A",
		Settings: map[string]interface{}{"search": map[string]interface{}{"default_top_k": 100}},
	})
	if err == nil {
		t.Fatal("expected error for invalid default_top_k")
	}
	if !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestService_Create_Conflict(t *testing.T) {
	svc := &Service{Repo: &serviceTestRepo{createErr: ErrConflict}}
	_, err := svc.Create(context.Background(), AppCreate{Name: "Dup"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_GetSettings_NotFound(t *testing.T) {
	svc := &Service{Repo: &serviceTestRepo{getSettingsErr: ErrNotFound}}
	_, err := svc.GetSettings(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected not found")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateSettings_ValidationError(t *testing.T) {
	svc := &Service{Repo: &serviceTestRepo{settings: map[string]interface{}{}}}
	_, err := svc.UpdateSettings(context.Background(), "some-id", map[string]interface{}{
		"search": map[string]interface{}{"confidence_threshold": 1.5},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

type serviceTestRepo struct {
	createErr     error
	getSettingsErr error
	settings      map[string]interface{}
}

func (m *serviceTestRepo) Create(ctx context.Context, a *App) error { return m.createErr }
func (m *serviceTestRepo) GetByID(ctx context.Context, id string) (*App, error) {
	return nil, nil
}
func (m *serviceTestRepo) List(ctx context.Context) ([]App, error) { return nil, nil }
func (m *serviceTestRepo) Update(ctx context.Context, id string, u *AppUpdate) (*App, error) {
	return nil, nil
}
func (m *serviceTestRepo) GetSettings(ctx context.Context, appID string) (map[string]interface{}, error) {
	if m.getSettingsErr != nil {
		return nil, m.getSettingsErr
	}
	return m.settings, nil
}
func (m *serviceTestRepo) UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error {
	return nil
}
