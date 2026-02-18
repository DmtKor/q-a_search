package auth

import "testing"

func TestParseBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{"empty", "", "", false},
		{"only Bearer", "Bearer", "", false},
		{"Bearer no space token", "Bearertoken", "", false},
		{"lowercase bearer", "bearer mytoken", "mytoken", true},
		{"uppercase Bearer", "Bearer mytoken", "mytoken", true},
		{"mixed Bearer", "BeArEr mytoken", "mytoken", true},
		{"extra spaces", "  Bearer   mytoken  ", "mytoken", true},
		{"token with spaces", "Bearer foo bar", "foo bar", true},
		{"tab after Bearer", "Bearer\tmytoken", "mytoken", true},
		{"no space after Bearer", "Bearermytoken", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseBearerToken(tt.header)
			if ok != tt.wantOK || got != tt.wantToken {
				t.Errorf("ParseBearerToken(%q) = (%q, %v), want (%q, %v)", tt.header, got, ok, tt.wantToken, tt.wantOK)
			}
		})
	}
}
