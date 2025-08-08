package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"

	"github.com/teamwork/mcp/internal/config"
)

// knownRootFields contains the default list of top-level JSON fields that
// should be ignored when processing entities for web link injection. These
// fields typically contain metadata rather than entity data.
var knownRootFields = []string{
	"meta",
	"included",
}

// WebLinkerOptions holds configuration options for the WebLinker function.
type WebLinkerOptions struct {
	// ignoreFields specifies which top-level JSON fields should be skipped when
	// processing entities for web link injection.
	ignoreFields []string
}

// WebLinkerOption is a function that configures the WebLinkerOptions.
type WebLinkerOption func(*WebLinkerOptions)

// WebLinkerWithIgnoreFields creates an option to specify additional fields that
// should be ignored when processing JSON data for web link injection. These
// fields will be skipped in addition to the default knownRootFields ("meta" and
// "included").
func WebLinkerWithIgnoreFields(fields ...string) WebLinkerOption {
	return func(opts *WebLinkerOptions) {
		opts.ignoreFields = fields
	}
}

// WebLinker processes JSON data to inject web links into entities based on
// their structure. It decodes the input data as JSON, traverses the top-level
// fields, and adds a "webLink" field in the meta section to qualifying objects
// using the provided buildPath function and customer URL from context.
//
// The function handles two types of structures for each top-level field:
//   - Single objects: {"field": {"id": 123, ...}} → adds webLink to the object
//   - Arrays of objects: {"field": [{"id": 123, ...}, ...]} → adds webLink to each object in the array
//
// Behavior:
//   - Returns original data unchanged if JSON parsing fails, customer URL is missing, or buildPath is nil
//   - Skips fields listed in the ignoreFields option (defaults to "meta" and "included")
//   - Only processes objects within arrays; non-object array items are left unchanged
//   - The webLink is constructed as: "{customerURL}/{path}" where path comes from buildPath()
//   - If buildPath returns an empty string for an object, no webLink is added to that object
//
// Parameters:
//   - ctx: Context containing customer URL via config.CustomerURLFromContext
//   - data: Raw JSON data as bytes
//   - buildPath: Function that generates a path string from an object (e.g., "#users/123")
//   - opts: Optional configuration (e.g., WebLinkerWithIgnoreFields to skip additional fields)
//
// Returns the modified JSON data as bytes, or the original data if processing
// fails.
func WebLinker(
	ctx context.Context,
	data []byte,
	buildPath func(map[string]any) string,
	opts ...WebLinkerOption,
) []byte {
	options := WebLinkerOptions{
		ignoreFields: knownRootFields,
	}
	for _, opt := range opts {
		opt(&options)
	}

	url, ok := config.CustomerURLFromContext(ctx)
	if !ok || url == "" || buildPath == nil {
		return data
	}
	url = strings.TrimSuffix(url, "/")

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return data
	}

	buildLink := func(object map[string]any) map[string]any {
		path := buildPath(object)
		if path == "" {
			return object
		}
		link := fmt.Sprintf("%s/%s", url, strings.TrimPrefix(path, "/"))
		if meta, ok := object["meta"]; ok {
			if m, ok := meta.(map[string]any); ok {
				if _, exists := m["webLink"]; exists {
					// If meta already has a webLink, do not overwrite it
					return object
				}
				m["webLink"] = link
				object["meta"] = m
			}
		} else {
			object["meta"] = map[string]any{
				"webLink": link,
			}
		}
		return object
	}

	for key, entity := range decoded {
		if slices.Contains(options.ignoreFields, key) {
			continue
		}
		switch v := entity.(type) {
		case map[string]any:
			decoded[key] = buildLink(v)
		case []any:
			for i, item := range v {
				if m, ok := item.(map[string]any); ok {
					v[i] = buildLink(m)
				}
			}
			decoded[key] = v
		}
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		return data
	}
	return encoded
}

// WebLinkerWithIDPathBuilder creates a path builder function for entities with
// an "id" field. It returns a function that builds a path in the format
// "prefix/id" for objects containing a non-zero "id" field. Returns an empty
// string if the "id" field is missing or has a zero value.
func WebLinkerWithIDPathBuilder(prefix string) func(map[string]any) string {
	return func(object map[string]any) string {
		id, ok := object["id"]
		if !ok {
			return ""
		}
		if id == reflect.Zero(reflect.TypeOf(id)).Interface() {
			return ""
		}
		// round float64 IDs to int64 to avoid decimal points in URLs
		if numeric, ok := id.(float64); ok && math.Trunc(numeric) == numeric {
			id = int64(numeric)
		}
		return fmt.Sprintf("%s/%v", prefix, id)
	}
}
