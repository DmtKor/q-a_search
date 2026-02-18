package auth

import "testing"

func TestIsOwner(t *testing.T) {
	tests := []struct {
		name       string
		principal  *Principal
		createdBy  string
		wantOwner  bool
	}{
		{"nil principal", nil, "uuid-1", false},
		{"empty created_by", &Principal{TokenID: "uuid-1"}, "", false},
		{"match", &Principal{TokenID: "uuid-1"}, "uuid-1", true},
		{"match lowercase", &Principal{TokenID: "UUID-1"}, "uuid-1", true},
		{"match spaces", &Principal{TokenID: "  uuid-1  "}, "uuid-1", true},
		{"mismatch", &Principal{TokenID: "uuid-1"}, "uuid-2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsOwner(tt.principal, tt.createdBy)
			if got != tt.wantOwner {
				t.Errorf("IsOwner() = %v, want %v", got, tt.wantOwner)
			}
		})
	}
}
