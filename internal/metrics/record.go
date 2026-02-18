package metrics

// Record is a single request_metrics row (created_at is set by DB).
type Record struct {
	Endpoint       string  // raw path, e.g. r.URL.Path
	StatusCode     int     // HTTP status code
	ResponseTimeMs int     // elapsed milliseconds
	TokenID        *string // UUID or nil when unauthenticated
	AppID          *string // UUID when token_type is app, else nil
}
