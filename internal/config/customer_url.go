package config

import "context"

type customerURLKey struct{}

// WithCustomerURL returns a new context with the given customer URL.
func WithCustomerURL(ctx context.Context, customerURL string) context.Context {
	return context.WithValue(ctx, customerURLKey{}, customerURL)
}

// CustomerURLFromContext returns the customer URL from the context, if any.
func CustomerURLFromContext(ctx context.Context) (string, bool) {
	customerURL, ok := ctx.Value(customerURLKey{}).(string)
	return customerURL, ok
}
