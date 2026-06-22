package helpers

import (
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// WithDateTypeSchema registers a JSON-schema override for the twapi.Date type
// on the given generation options. twapi.Date is defined as `type Date
// time.Time`, so the reflection-based generator would otherwise emit a useless
// object schema for time.Time's unexported fields. This override forces it to a
// nullable, date-only string.
//
// Use it whenever generating an output schema from a response type that carries
// (or sideloads) twapi.Date fields:
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
	return opts
}
