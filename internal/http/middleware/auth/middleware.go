package auth

import (
	"net/http"
	"time"

	"github.com/yourusername/project/internal/auth"
)

// Authenticate returns a middleware that parses Authorization: Bearer <token>,
// verifies it via TokenStore, puts principal in context, and on success always calls UpdateLastUsedAt.
// Secret is used to compute token hash (HMAC-SHA256). Returns 401 on missing/invalid/expired/disabled token.
func Authenticate(store auth.TokenStore, secret []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, ok := auth.ParseBearerToken(r.Header.Get("Authorization"))
			if !ok {
				Write401(w, "Missing or invalid authorization token")
				return
			}
			hash := auth.HashToken(secret, rawToken)
			row, err := store.GetByTokenHash(r.Context(), hash)
			if err != nil {
				Write401(w, "Missing or invalid authorization token")
				return
			}
			if row == nil {
				Write401(w, "Missing or invalid authorization token")
				return
			}
			if !row.IsActive {
				Write401(w, "Missing or invalid authorization token")
				return
			}
			if row.ExpiresAt != nil && row.ExpiresAt.Before(time.Now()) {
				Write401(w, "Missing or invalid authorization token")
				return
			}
			principal := &auth.Principal{
				TokenID:   row.ID,
				TokenType: row.TokenType,
				Role:      row.Role,
				AppID:     row.AppID,
			}
			ctx := auth.WithPrincipal(r.Context(), principal)
			// Always update last_used_at on successful auth (fire-and-forget or best-effort)
			_ = store.UpdateLastUsedAt(r.Context(), row.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAppOrStaff wraps next and returns 403 if there is no principal or token type is not "app" or "staff".
func RequireAppOrStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := auth.PrincipalFromContext(r.Context())
		if p == nil {
			Write403(w, "Access denied")
			return
		}
		if p.TokenType != "app" && p.TokenType != "staff" {
			Write403(w, "Access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireStaff wraps next and returns 403 if there is no principal or token type is not "staff".
func RequireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := auth.PrincipalFromContext(r.Context())
		if p == nil {
			Write403(w, "Access denied")
			return
		}
		if p.TokenType != "staff" {
			Write403(w, "Access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}
