package config

import "context"

type scopesKey struct{}

// WithScopes adds all scopes related to the Bearer Token to the context.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesKey{}, scopes)
}

func scopes(ctx context.Context) []string {
	scopes, ok := ctx.Value(scopesKey{}).([]string)
	if !ok {
		return nil
	}
	return scopes
}
