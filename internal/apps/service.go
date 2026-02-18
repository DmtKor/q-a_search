package apps

import (
	"context"
	"errors"
)

// Service implements business logic for apps and settings.
type Service struct {
	Repo Repository
}

// Repository is the persistence interface (implemented by repository.Pool).
type Repository interface {
	Create(ctx context.Context, a *App) error
	GetByID(ctx context.Context, id string) (*App, error)
	List(ctx context.Context) ([]App, error)
	Update(ctx context.Context, id string, u *AppUpdate) (*App, error)
	GetSettings(ctx context.Context, appID string) (map[string]interface{}, error)
	UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error
}

// Create creates an app. Returns ErrConflict if name already exists, ErrValidation if body invalid.
func (s *Service) Create(ctx context.Context, body AppCreate) (*App, error) {
	if body.Name == "" {
		return nil, ErrValidation
	}
	if err := ValidateSettings(body.Settings); err != nil {
		return nil, err
	}
	a := &App{
		Name:     body.Name,
		Settings: body.Settings,
	}
	if a.Settings == nil {
		a.Settings = make(map[string]interface{})
	}
	if err := s.Repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// List returns all apps.
func (s *Service) List(ctx context.Context) ([]App, error) {
	return s.Repo.List(ctx)
}

// Get returns an app by ID. Returns ErrNotFound if not found.
func (s *Service) Get(ctx context.Context, id string) (*App, error) {
	return s.Repo.GetByID(ctx, id)
}

// Update updates an app. Returns ErrNotFound, ErrConflict (duplicate name), or ErrValidation.
func (s *Service) Update(ctx context.Context, id string, body AppUpdate) (*App, error) {
	if body.Name != nil && *body.Name == "" {
		return nil, ErrValidation
	}
	if body.Settings != nil {
		if err := ValidateSettings(body.Settings); err != nil {
			return nil, err
		}
	}
	return s.Repo.Update(ctx, id, &body)
}

// GetSettings returns settings for an app. Returns ErrNotFound if app not found.
func (s *Service) GetSettings(ctx context.Context, appID string) (map[string]interface{}, error) {
	return s.Repo.GetSettings(ctx, appID)
}

// UpdateSettings replaces settings for an app (atomic). Returns ErrNotFound or ErrValidation.
func (s *Service) UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) (map[string]interface{}, error) {
	if settings == nil {
		settings = make(map[string]interface{})
	}
	if err := ValidateSettings(settings); err != nil {
		return nil, err
	}
	if err := s.Repo.UpdateSettings(ctx, appID, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// Export returns settings for an app (same as GetSettings). For round-trip with Import.
func (s *Service) Export(ctx context.Context, appID string) (map[string]interface{}, error) {
	return s.Repo.GetSettings(ctx, appID)
}

// Import replaces settings from JSON (same as UpdateSettings). Returns ErrNotFound or ErrValidation.
func (s *Service) Import(ctx context.Context, appID string, settings map[string]interface{}) (map[string]interface{}, error) {
	return s.UpdateSettings(ctx, appID, settings)
}

// IsValidationError returns true if err is from validation (422).
func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidation)
}
