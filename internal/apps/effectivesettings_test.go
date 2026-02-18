package apps

import (
	"context"
	"testing"

	"github.com/yourusername/project/internal/auth"
)

func TestEffectiveSettingsResolver_Staff_ReturnsDefaults(t *testing.T) {
	repo := &mockRepo{settings: nil}
	resolver := &EffectiveSettingsResolver{Repo: repo}
	principal := &auth.Principal{TokenType: "staff"}
	th, k, err := resolver.GetEffectiveSearchSettings(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if th != DefaultThreshold || k != DefaultTopK {
		t.Errorf("staff: got threshold=%v topK=%v, want %v %v", th, k, DefaultThreshold, DefaultTopK)
	}
	if repo.getSettingsCalled {
		t.Error("staff should not call GetSettings")
	}
}

func TestEffectiveSettingsResolver_NilPrincipal_ReturnsDefaults(t *testing.T) {
	repo := &mockRepo{}
	resolver := &EffectiveSettingsResolver{Repo: repo}
	th, k, err := resolver.GetEffectiveSearchSettings(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if th != DefaultThreshold || k != DefaultTopK {
		t.Errorf("nil principal: got threshold=%v topK=%v", th, k)
	}
}

func TestEffectiveSettingsResolver_AppToken_LoadsSettings(t *testing.T) {
	appID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	repo := &mockRepo{
		settings: map[string]interface{}{
			"search": map[string]interface{}{
				"confidence_threshold": 0.9,
				"default_top_k":      15,
			},
		},
	}
	resolver := &EffectiveSettingsResolver{Repo: repo}
	principal := &auth.Principal{TokenType: "app", AppID: &appID}
	th, k, err := resolver.GetEffectiveSearchSettings(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if th != 0.9 || k != 15 {
		t.Errorf("app: got threshold=%v topK=%v, want 0.9 15", th, k)
	}
	if !repo.getSettingsCalled || repo.getSettingsAppID != appID {
		t.Errorf("GetSettings not called with app_id: called=%v appID=%q", repo.getSettingsCalled, repo.getSettingsAppID)
	}
}

func TestEffectiveSettingsResolver_AppToken_NotFound_ReturnsDefaults(t *testing.T) {
	appID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	repo := &mockRepo{err: ErrNotFound}
	resolver := &EffectiveSettingsResolver{Repo: repo}
	principal := &auth.Principal{TokenType: "app", AppID: &appID}
	th, k, err := resolver.GetEffectiveSearchSettings(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if th != DefaultThreshold || k != DefaultTopK {
		t.Errorf("app not found: got threshold=%v topK=%v", th, k)
	}
}

type mockRepo struct {
	settings          map[string]interface{}
	err               error
	getSettingsCalled bool
	getSettingsAppID  string
}

func (m *mockRepo) Create(ctx context.Context, a *App) error { return nil }
func (m *mockRepo) GetByID(ctx context.Context, id string) (*App, error) {
	return nil, nil
}
func (m *mockRepo) List(ctx context.Context) ([]App, error) {
	return nil, nil
}
func (m *mockRepo) Update(ctx context.Context, id string, u *AppUpdate) (*App, error) {
	return nil, nil
}
func (m *mockRepo) GetSettings(ctx context.Context, appID string) (map[string]interface{}, error) {
	m.getSettingsCalled = true
	m.getSettingsAppID = appID
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}
func (m *mockRepo) UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error {
	return nil
}
