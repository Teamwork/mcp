// Package twctx carries the per-request values the MCP server derives from the
// caller's bearer token: which installation it belongs to, the token itself,
// the scopes it grants, and whether the installation lives in another region.
//
// It exists as its own package so tool helpers can read those values without
// importing the server configuration, which would otherwise make every tool
// package depend on the whole startup path.
package twctx

import "context"

type (
	bearerTokenKey struct{}
	crossRegionKey struct{}
	customerURLKey struct{}
	scopesKey      struct{}
)

// WithBearerToken returns a new context with the given bearer token.
func WithBearerToken(ctx context.Context, bearerToken string) context.Context {
	return context.WithValue(ctx, bearerTokenKey{}, bearerToken)
}

// BearerTokenFromContext returns the bearer token from the context, if any.
func BearerTokenFromContext(ctx context.Context) (string, bool) {
	bearerToken, ok := ctx.Value(bearerTokenKey{}).(string)
	return bearerToken, ok
}

// WithCustomerURL returns a new context with the given customer URL.
func WithCustomerURL(ctx context.Context, customerURL string) context.Context {
	return context.WithValue(ctx, customerURLKey{}, customerURL)
}

// CustomerURLFromContext returns the customer URL from the context, if any.
func CustomerURLFromContext(ctx context.Context) (string, bool) {
	customerURL, ok := ctx.Value(customerURLKey{}).(string)
	return customerURL, ok
}

// WithCrossRegion adds a boolean value to the context indicating if the request
// is cross-region.
func WithCrossRegion(ctx context.Context, crossRegion bool) context.Context {
	return context.WithValue(ctx, crossRegionKey{}, crossRegion)
}

// IsCrossRegion reports whether the request targets an installation in another
// region.
func IsCrossRegion(ctx context.Context) bool {
	crossRegion, ok := ctx.Value(crossRegionKey{}).(bool)
	return ok && crossRegion
}

// WithScopes adds all scopes related to the bearer token to the context.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesKey{}, scopes)
}

// ScopesFromContext returns the scopes granted to the bearer token, if any.
func ScopesFromContext(ctx context.Context) []string {
	scopes, ok := ctx.Value(scopesKey{}).([]string)
	if !ok {
		return nil
	}
	return scopes
}
