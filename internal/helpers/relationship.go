package helpers

import (
	"encoding/json"
	"strconv"
)

// RelationshipMetaID reads a numeric ID out of a relationship's meta bag.
//
// The v3 API attaches hints to a relationship under "meta" — a task's tasklist
// relationship, for instance, carries the "projectId" the tasklist belongs to,
// which saves loading the tasklist just to resolve its project. The bag is
// decoded as map[string]any, so a JSON number arrives as float64; the other
// cases are covered for callers that decode with json.Number or receive the
// value as a string.
//
// It reports false when the key is absent, holds a non-numeric value, or is not
// a positive ID, so callers can fall back to whatever they did before the hint
// existed.
func RelationshipMetaID(meta map[string]any, key string) (int64, bool) {
	value, ok := meta[key]
	if !ok {
		return 0, false
	}

	var id int64
	switch v := value.(type) {
	case float64:
		id = int64(v)
	case int64:
		id = v
	case int:
		id = int64(v)
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		id = parsed
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		id = parsed
	default:
		return 0, false
	}

	if id <= 0 {
		return 0, false
	}
	return id, true
}
