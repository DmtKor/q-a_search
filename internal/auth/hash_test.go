package auth

import (
	"testing"
)

func TestHashToken(t *testing.T) {
	secret := []byte("test-secret")
	token := "raw-token"
	h1 := HashToken(secret, token)
	h2 := HashToken(secret, token)
	if h1 != h2 {
		t.Errorf("HashToken should be deterministic: %q != %q", h1, h2)
	}
	if h1 == "" {
		t.Error("HashToken should not return empty string")
	}
	// Different secret => different hash
	h3 := HashToken([]byte("other-secret"), token)
	if h1 == h3 {
		t.Error("HashToken with different secret should differ")
	}
	// Different token => different hash
	h4 := HashToken(secret, "other-token")
	if h1 == h4 {
		t.Error("HashToken with different token should differ")
	}
}
