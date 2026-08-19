package helpers

import (
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// WithDateTypeSchema registers JSON-schema overrides for the SDK's date types
// on the given generation options. Both twapi.Date and twapi.OptionalDateTime
// are defined over time.Time (`type Date time.Time`), so the reflection-based
// generator would otherwise emit a useless object schema for time.Time's
// unexported fields — while their MarshalJSON methods emit a string. Every
// response carrying one then fails output-schema validation on every call. The
// overrides force them to nullable strings matching what the marshaller
// actually writes.
//
// The types are covered here rather than at each call site because a response
// picks them up transitively: projects.Team is the only model with an
// OptionalDateTime field, but SearchResponse sideloads Team, so the search
// schema needs the same override.
//
// Use it whenever generating an output schema from a response type that carries
// (or sideloads) those fields:
//
//	schema, err = jsonschema.For[Response](helpers.WithDateTypeSchema(&jsonschema.ForOptions{}))
//
// Any other options already set on opts (including pre-existing TypeSchemas
// entries) are preserved. The options value is modified in place and also
// returned for convenient chaining.
func WithDateTypeSchema(opts *jsonschema.ForOptions) *jsonschema.ForOptions {
	if opts == nil {
		opts = &jsonschema.ForOptions{}
	}
	if opts.TypeSchemas == nil {
		opts.TypeSchemas = make(map[reflect.Type]*jsonschema.Schema)
	}
	opts.TypeSchemas[reflect.TypeFor[twapi.Date]()] = &jsonschema.Schema{
		Types:       []string{"null", "string"},
		Format:      "date",
		Description: "Null or date-only date string",
	}
	opts.TypeSchemas[reflect.TypeFor[twapi.OptionalDateTime]()] = &jsonschema.Schema{
		Types:       []string{"null", "string"},
		Format:      "date-time",
		Description: "Null or RFC3339 date-time string. Null when the value is unset.",
	}
	return opts
}
