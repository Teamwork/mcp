package helpers

// SliceToAny converts a slice of any type to []any for use with filter.In()
func SliceToAny[T any](slice []T) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
