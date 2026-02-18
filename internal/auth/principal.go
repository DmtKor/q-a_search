package auth

import "context"

// Principal holds the authenticated token identity for the request.
type Principal struct {
	TokenID   string // UUID from auth_tokens.id (used for IsOwner vs cases.created_by)
	TokenType string // "app" or "staff"
	Role      string
	AppID     *string // set when token_type is "app"
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var principalKey = contextKey{name: "principal"}

// WithPrincipal returns a context with the principal attached.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFromContext returns the principal from the context, or nil if not set.
func PrincipalFromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey).(*Principal)
	return p
}
