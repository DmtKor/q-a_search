package auth

import (
	"strings"
	"unicode"
)

// BearerPrefix is the expected prefix in Authorization header (case-insensitive), without trailing space.
const BearerPrefix = "bearer"

// ParseBearerToken extracts the raw token from "Authorization: Bearer <token>".
// It accepts any casing of "Bearer" and any single whitespace (space, tab) after it; trims token. Returns (token, true) if valid, ("", false) otherwise.
func ParseBearerToken(authHeader string) (rawToken string, ok bool) {
	s := strings.TrimSpace(authHeader)
	if len(s) <= len(BearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(s[:len(BearerPrefix)], BearerPrefix) {
		return "", false
	}
	rest := s[len(BearerPrefix):]
	if len(rest) == 0 || !unicode.IsSpace(rune(rest[0])) {
		return "", false
	}
	token := strings.TrimLeftFunc(rest, unicode.IsSpace)
	if token == "" {
		return "", false
	}
	return token, true
}
