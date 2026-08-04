package helpers

import (
	"encoding/json"
	"fmt"
)

// ProjectCollectionFields narrows every object in the named top-level
// collection of a v3 list response down to keep, dropping all other
// attributes, and returns the re-encoded body.
//
// This exists for endpoints that do not implement sparse fieldsets. Sparse
// fieldset support is opt-in per endpoint
// (https://apidocs.teamwork.com/guides/teamwork/sparse-fieldsets), so on an
// endpoint that ignores `fields[...]` the server returns every attribute no
// matter what the request asked for, and a `verbose=false` that only sets
// Filters.Fields is silently a no-op. Projecting here guarantees the caller
// actually gets the smaller payload.
//
// Prefer real sparse fieldsets when the endpoint supports them: they save the
// bandwidth between us and the API, this only saves the tokens between us and
// the MCP client. Everything outside collection — notably meta and included —
// is passed through untouched, and a body whose collection is missing or is
// not an array of objects is returned unchanged rather than treated as an
// error, so an unexpected shape degrades to verbose output instead of failing
// the call.
func ProjectCollectionFields(body []byte, collection string, keep []string) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response envelope: %w", err)
	}

	raw, ok := envelope[collection]
	if !ok {
		return body, nil
	}

	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		// Not an array of objects; leave the payload alone.
		return body, nil //nolint:nilerr // an unexpected shape falls back to the full body
	}

	allowed := make(map[string]struct{}, len(keep))
	for _, field := range keep {
		allowed[field] = struct{}{}
	}

	projected := make([]map[string]json.RawMessage, 0, len(items))
	for _, item := range items {
		trimmed := make(map[string]json.RawMessage, len(allowed))
		for field, value := range item {
			if _, ok := allowed[field]; ok {
				trimmed[field] = value
			}
		}
		projected = append(projected, trimmed)
	}

	encoded, err := json.Marshal(projected)
	if err != nil {
		return nil, fmt.Errorf("failed to encode projected %s: %w", collection, err)
	}
	envelope[collection] = encoded

	result, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response envelope: %w", err)
	}
	return result, nil
}
