package apps

import (
	"errors"
	"fmt"
)

const (
	MinTopK          = 1
	MaxTopK          = 50
	MinThreshold     = 0.0
	MaxThreshold     = 1.0
	DefaultThreshold = 0.7
	DefaultTopK      = 10
)

// ErrValidation is returned when settings fail validation (422).
var ErrValidation = errors.New("settings validation error")

// ValidateSettings checks search.default_top_k (1..50) and search.confidence_threshold (0..1).
// Other keys in settings are allowed (additionalProperties). Returns ErrValidation with message.
func ValidateSettings(settings map[string]interface{}) error {
	if settings == nil {
		return nil
	}
	searchVal, ok := settings["search"]
	if !ok {
		return nil
	}
	searchMap, ok := searchVal.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w: search must be an object", ErrValidation)
	}
	if v, ok := searchMap["default_top_k"]; ok {
		switch n := v.(type) {
		case float64:
			k := int(n)
			if float64(k) != n || k < MinTopK || k > MaxTopK {
				return fmt.Errorf("%w: search.default_top_k must be integer between %d and %d", ErrValidation, MinTopK, MaxTopK)
			}
		case int:
			if n < MinTopK || n > MaxTopK {
				return fmt.Errorf("%w: search.default_top_k must be between %d and %d", ErrValidation, MinTopK, MaxTopK)
			}
		default:
			return fmt.Errorf("%w: search.default_top_k must be an integer", ErrValidation)
		}
	}
	if v, ok := searchMap["confidence_threshold"]; ok {
		f, ok := toFloat64(v)
		if !ok {
			return fmt.Errorf("%w: search.confidence_threshold must be a number", ErrValidation)
		}
		if f < MinThreshold || f > MaxThreshold {
			return fmt.Errorf("%w: search.confidence_threshold must be between %.1f and %.1f", ErrValidation, MinThreshold, MaxThreshold)
		}
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// EffectiveSearchSettings returns threshold and defaultTopK from settings, or defaults if missing/invalid.
func EffectiveSearchSettings(settings map[string]interface{}) (threshold float64, defaultTopK int) {
	threshold = DefaultThreshold
	defaultTopK = DefaultTopK
	if settings == nil {
		return threshold, defaultTopK
	}
	searchVal, ok := settings["search"]
	if !ok {
		return threshold, defaultTopK
	}
	searchMap, ok := searchVal.(map[string]interface{})
	if !ok {
		return threshold, defaultTopK
	}
	if v, ok := searchMap["confidence_threshold"]; ok {
		if f, ok := toFloat64(v); ok && f >= MinThreshold && f <= MaxThreshold {
			threshold = f
		}
	}
	if v, ok := searchMap["default_top_k"]; ok {
		switch n := v.(type) {
		case float64:
			k := int(n)
			if float64(k) == n && k >= MinTopK && k <= MaxTopK {
				defaultTopK = k
			}
		case int:
			if n >= MinTopK && n <= MaxTopK {
				defaultTopK = n
			}
		}
	}
	return threshold, defaultTopK
}
