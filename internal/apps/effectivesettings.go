package apps

import (
	"context"

	"github.com/yourusername/project/internal/auth"
	"github.com/yourusername/project/internal/search"
)

// EffectiveSettingsResolver implements search.AppSettingsReader.
// Returns effective threshold and defaultTopK for the principal (app_id → settings or defaults for staff).
// Glue wires this into search.Service as AppSettings.
type EffectiveSettingsResolver struct {
	Repo Repository
}

// GetEffectiveSearchSettings returns threshold and defaultTopK for the current principal.
// For app token: loads app settings by principal.AppID and applies EffectiveSearchSettings.
// For staff or missing app: returns DefaultThreshold and DefaultTopK from this package.
func (r *EffectiveSettingsResolver) GetEffectiveSearchSettings(ctx context.Context, principal *auth.Principal) (threshold float64, defaultTopK int, err error) {
	if principal == nil {
		return DefaultThreshold, DefaultTopK, nil
	}
	if principal.TokenType == "app" && principal.AppID != nil && *principal.AppID != "" {
		settings, err := r.Repo.GetSettings(ctx, *principal.AppID)
		if err != nil {
			if err == ErrNotFound {
				return DefaultThreshold, DefaultTopK, nil
			}
			return 0, 0, err
		}
		threshold, defaultTopK = EffectiveSearchSettings(settings)
		return threshold, defaultTopK, nil
	}
	return DefaultThreshold, DefaultTopK, nil
}

// Ensure EffectiveSettingsResolver implements search.AppSettingsReader.
var _ search.AppSettingsReader = (*EffectiveSettingsResolver)(nil)
