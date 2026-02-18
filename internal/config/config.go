package config

import (
	"os"
	"strconv"
)

// LogLevel for request/response logging: "none", "minimal" (method, path, status, duration), "detailed" (+ request/response size, optional body snippet).
const (
	LogLevelNone    = "none"
	LogLevelMinimal = "minimal"
	LogLevelDetailed = "detailed"
)

// Config holds application configuration (wiring only; no business logic).
type Config struct {
	DSN                  string
	Secret               []byte
	TicketsBaseURL       string
	TemplateMaxOutputLen int
	// RequestLogLevel: none | minimal | detailed (env REQUEST_LOG_LEVEL or LOG_LEVEL).
	RequestLogLevel string
}

// Load reads config from environment with defaults.
// DSN: DATABASE_URL or POSTGRES_DSN; default "" (must be set).
// Secret: AUTH_SECRET or HMAC_SECRET; default "dev-secret-change-in-production".
// TicketsBaseURL: TICKETS_BASE_URL; default "/api/v1/tickets".
// TemplateMaxOutputLen: TEMPLATE_MAX_OUTPUT_LEN; default 32000.
func Load() *Config {
	c := &Config{
		DSN:                  getEnv("DATABASE_URL", getEnv("POSTGRES_DSN", "")),
		Secret:               []byte(getEnv("AUTH_SECRET", getEnv("HMAC_SECRET", "dev-secret-change-in-production"))),
		TicketsBaseURL:       getEnv("TICKETS_BASE_URL", "/api/v1/tickets"),
		TemplateMaxOutputLen: 32000,
		RequestLogLevel:      getEnv("REQUEST_LOG_LEVEL", getEnv("LOG_LEVEL", LogLevelMinimal)),
	}
	if n := getEnv("TEMPLATE_MAX_OUTPUT_LEN", ""); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			c.TemplateMaxOutputLen = v
		}
	}
	switch c.RequestLogLevel {
	case LogLevelNone, LogLevelMinimal, LogLevelDetailed:
		// ok
	default:
		c.RequestLogLevel = LogLevelMinimal
	}
	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
