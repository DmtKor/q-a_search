package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/yourusername/project/internal/auth"
	pkgmetrics "github.com/yourusername/project/internal/metrics"
)

type contextKey struct{ name string }

var carrierKey = contextKey{name: "metrics_principal_carrier"}

// principalCarrier holds the principal for the request so EnrichPrincipal can set it
// and Metrics can read it after the chain returns (request context is unchanged).
type principalCarrier struct {
	Principal *auth.Principal
}

// Metrics returns a middleware that records endpoint, status_code, response_time_ms,
// and (when EnrichPrincipal runs after Auth) token_id/app_id. Uses raw path r.URL.Path as endpoint.
// Chain order must be: Metrics(writer)(Auth(...)(EnrichPrincipal(Handlers))).
// Write errors are ignored (best-effort).
func Metrics(writer pkgmetrics.Writer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			carrier := &principalCarrier{}
			r = r.WithContext(withCarrier(r.Context(), carrier))
			wrapped := newResponseWriter(w)
			start := time.Now()
			next.ServeHTTP(wrapped, r)
			elapsedMs := int(time.Since(start).Milliseconds())
			rec := &pkgmetrics.Record{
				Endpoint:       r.URL.Path,
				StatusCode:     wrapped.Status(),
				ResponseTimeMs: elapsedMs,
				TokenID:        nil,
				AppID:          nil,
			}
			if carrier.Principal != nil {
				rec.TokenID = strPtr(carrier.Principal.TokenID)
				rec.AppID = carrier.Principal.AppID
			}
			_ = writer.Write(r.Context(), rec) // best-effort, ignore error
		})
	}
}

func withCarrier(ctx context.Context, c *principalCarrier) context.Context {
	return context.WithValue(ctx, carrierKey, c)
}

func strPtr(s string) *string { return &s }

// EnrichPrincipal must run after Auth. It copies the principal from context into the
// carrier so that Metrics (which runs before Auth) can read token_id/app_id after the chain returns.
// Order: Metrics(writer)(Auth(...)(EnrichPrincipal(Handlers))).
func EnrichPrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := carrierFromContext(r.Context()); c != nil {
			c.Principal = auth.PrincipalFromContext(r.Context())
		}
		next.ServeHTTP(w, r)
	})
}

func carrierFromContext(ctx context.Context) *principalCarrier {
	v := ctx.Value(carrierKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*principalCarrier)
	return c
}