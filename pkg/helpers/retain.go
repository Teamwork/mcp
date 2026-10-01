package helpers

// KeySet is a set of JSON attribute names.
type KeySet map[string]struct{}

// NewKeySet builds a KeySet from names.
func NewKeySet[S ~string](names ...S) KeySet {
	set := make(KeySet, len(names))
	for _, name := range names {
		set[string(name)] = struct{}{}
	}
	return set
}

// Has reports whether name is in the set.
func (s KeySet) Has(name string) bool {
	_, ok := s[name]
	return ok
}

// RetainKeys deletes every attribute of a decoded JSON object that keep does
// not list, so a response carries only the attributes the server chose to
// publish. A value that is not an object is left alone.
func RetainKeys(value any, keep KeySet) {
	obj, ok := value.(map[string]any)
	if !ok {
		return
	}
	for key := range obj {
		if !keep.Has(key) {
			delete(obj, key)
		}
	}
}

// RetainKeysEach applies RetainKeys to every element of a decoded JSON array,
// or to every value of a decoded JSON object keyed by ID (the shape of a v3
// included section).
func RetainKeysEach(value any, keep KeySet) {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			RetainKeys(item, keep)
		}
	case map[string]any:
		for _, item := range v {
			RetainKeys(item, keep)
		}
	}
}
