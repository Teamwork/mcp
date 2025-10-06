package helpers

// SliceToAny converts a slice of any type to []any for use with filter.In()
func SliceToAny[T any](slice []T) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

// IntSliceToInt64 converts a slice of int to a slice of int64
func IntSliceToInt64(slice []int) []int64 {
	result := make([]int64, len(slice))
	for i, v := range slice {
		result[i] = int64(v)
	}
	return result
}
