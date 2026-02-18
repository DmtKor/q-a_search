package http

import (
	nethttp "net/http"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	corsmw "github.com/yourusername/project/internal/http/middleware/cors"
	metricsmw "github.com/yourusername/project/internal/http/middleware/metrics"
	requestlogmw "github.com/yourusername/project/internal/http/middleware/requestlog"
	"github.com/yourusername/project/internal/metrics"

	caseshttp "github.com/yourusername/project/internal/cases/http"
	appshandler "github.com/yourusername/project/internal/apps/http"
	searchhandler "github.com/yourusername/project/internal/search/http"
	ticketshttp "github.com/yourusername/project/internal/tickets/http"
)

// RouterConfig holds dependencies for building the HTTP handler chain.
type RouterConfig struct {
	MetricsWriter    metrics.Writer
	TokenStore       auth.TokenStore
	Secret           []byte
	RequestLogLevel  string // none | minimal | detailed

	SearchHandler  *searchhandler.Handler
	CasesHandler   *caseshttp.Handler
	TicketsHandler *ticketshttp.Handler
	AppsHandler    *appshandler.Handler
}

// Handler returns the full middleware chain and mux: Metrics -> Auth -> EnrichPrincipal -> routes.
// Routes: POST /api/v1/search (RequireAppOrStaff); /api/v1/cases, /api/v1/tickets, /api/v1/apps (RequireStaff).
func Handler(cfg RouterConfig) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.Handle("POST /api/v1/search", authmw.RequireAppOrStaff(cfg.SearchHandler))
	mux.Handle("/api/v1/cases", authmw.RequireStaff(cfg.CasesHandler))
	mux.Handle("/api/v1/cases/", authmw.RequireStaff(cfg.CasesHandler))
	mux.Handle("/api/v1/tickets", authmw.RequireStaff(cfg.TicketsHandler))
	mux.Handle("/api/v1/tickets/", authmw.RequireStaff(cfg.TicketsHandler))
	mux.Handle("/api/v1/apps", authmw.RequireStaff(cfg.AppsHandler))
	mux.Handle("/api/v1/apps/", authmw.RequireStaff(cfg.AppsHandler))

	withPrincipal := metricsmw.EnrichPrincipal(mux)
	withAuth := authmw.Authenticate(cfg.TokenStore, cfg.Secret)(withPrincipal)
	withMetrics := metricsmw.Metrics(cfg.MetricsWriter)(withAuth)
	withRequestLog := requestlogmw.Middleware(cfg.RequestLogLevel, withMetrics)
	return corsmw.Middleware(withRequestLog)
}
