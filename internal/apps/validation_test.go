package apps

import (
	"errors"
	"testing"
)

func TestValidateSettings_EmptyOrNil(t *testing.T) {
	if err := ValidateSettings(nil); err != nil {
		t.Errorf("nil settings should be valid: %v", err)
	}
	if err := ValidateSettings(map[string]interface{}{}); err != nil {
		t.Errorf("empty settings should be valid: %v", err)
	}
}

func TestValidateSettings_DefaultTopK(t *testing.T) {
	tests := []struct {
		name    string
		val     interface{}
		wantErr bool
	}{
		{"valid 1", 1, false},
		{"valid 50", 50, false},
		{"valid float 10", float64(10), false},
		{"zero", 0, true},
		{"51", 51, true},
		{"float 0.5", 0.5, true},
		{"string", "10", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := map[string]interface{}{
				"search": map[string]interface{}{"default_top_k": tt.val},
			}
			err := ValidateSettings(settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSettings() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation wrap, got %v", err)
			}
		})
	}
}

func TestValidateSettings_ConfidenceThreshold(t *testing.T) {
	tests := []struct {
		name    string
		val     interface{}
		wantErr bool
	}{
		{"0", 0.0, false},
		{"1", 1.0, false},
		{"0.7", 0.7, false},
		{"-0.1", -0.1, true},
		{"1.1", 1.1, true},
		{"int 1", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := map[string]interface{}{
				"search": map[string]interface{}{"confidence_threshold": tt.val},
			}
			err := ValidateSettings(settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSettings() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSettings_SearchNotObject(t *testing.T) {
	settings := map[string]interface{}{"search": "not-an-object"}
	if err := ValidateSettings(settings); err == nil {
		t.Error("search as string should be invalid")
	}
}

func TestEffectiveSearchSettings_Defaults(t *testing.T) {
	th, k := EffectiveSearchSettings(nil)
	if th != DefaultThreshold || k != DefaultTopK {
		t.Errorf("nil: got threshold=%v topK=%v, want %v %v", th, k, DefaultThreshold, DefaultTopK)
	}
	th, k = EffectiveSearchSettings(map[string]interface{}{})
	if th != DefaultThreshold || k != DefaultTopK {
		t.Errorf("empty: got threshold=%v topK=%v", th, k)
	}
}

func TestEffectiveSearchSettings_FromSettings(t *testing.T) {
	settings := map[string]interface{}{
		"search": map[string]interface{}{
			"confidence_threshold": 0.85,
			"default_top_k":       20,
		},
	}
	th, k := EffectiveSearchSettings(settings)
	if th != 0.85 || k != 20 {
		t.Errorf("got threshold=%v topK=%v, want 0.85 20", th, k)
	}
}

func TestEffectiveSearchSettings_Partial(t *testing.T) {
	// only threshold set
	settings := map[string]interface{}{
		"search": map[string]interface{}{"confidence_threshold": 0.5},
	}
	th, k := EffectiveSearchSettings(settings)
	if th != 0.5 || k != DefaultTopK {
		t.Errorf("got threshold=%v topK=%v, want 0.5 %v", th, k, DefaultTopK)
	}
	// only top_k set
	settings2 := map[string]interface{}{
		"search": map[string]interface{}{"default_top_k": 5},
	}
	th2, k2 := EffectiveSearchSettings(settings2)
	if th2 != DefaultThreshold || k2 != 5 {
		t.Errorf("got threshold=%v topK=%v, want %v 5", th2, k2, DefaultThreshold)
	}
}
