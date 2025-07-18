package config

import "context"

type crossRegionKey struct{}

// WithCrossRegion adds a boolean value to the context indicating if the request
// is cross-region.
func WithCrossRegion(ctx context.Context, crossRegion bool) context.Context {
	return context.WithValue(ctx, crossRegionKey{}, crossRegion)
}

func isCrossRegion(ctx context.Context) bool {
	crossRegion, ok := ctx.Value(crossRegionKey{}).(bool)
	return ok && crossRegion
}
