package cors

import (
	"net/http"
	"strings"
)

// AllowedOrigins for CORS. Empty means no CORS; "*" allows any origin (dev only).
// Typical dev: []string{"http://localhost:5173", "http://127.0.0.1:5173"} or []string{"*"}.
var AllowedOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}

const (
	allowOrigin  = "Access-Control-Allow-Origin"
	allowMethods = "Access-Control-Allow-Methods"
	allowHeaders = "Access-Control-Allow-Headers"
	maxAge       = "Access-Control-Max-Age"
)

// Middleware adds CORS headers and handles OPTIONS preflight.
// Place at the outer layer of the chain (first to run).
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowOriginFor(origin) {
			w.Header().Set(allowOrigin, origin)
		}
		w.Header().Set(allowMethods, "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set(allowHeaders, "Authorization, Content-Type")
		w.Header().Set(maxAge, "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func allowOriginFor(origin string) bool {
	for _, o := range AllowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	// Allow any localhost with common dev ports
	if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
		return true
	}
	return false
}
