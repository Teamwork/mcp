package config

import "context"

type bearerTokenKey struct{}

// WithBearerToken returns a new context with the given bearer token.
func WithBearerToken(ctx context.Context, bearerToken string) context.Context {
	return context.WithValue(ctx, bearerTokenKey{}, bearerToken)
}

// BearerTokenFromContext returns the bearer token from the context, if any.
func BearerTokenFromContext(ctx context.Context) (string, bool) {
	bearerToken, ok := ctx.Value(bearerTokenKey{}).(string)
	return bearerToken, ok
}
