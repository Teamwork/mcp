package helpers

import "github.com/google/jsonschema-go/jsonschema"

// WithOptionalFields recursively relaxes a schema and every nested schema it
// references (Properties, Items, AdditionalProperties, AnyOf/OneOf/AllOf
// branches) so it can validate sparse or forward-compatible payloads:
//
//   - clears the `required` array, so sparse responses that omit fields still
//     validate;
//   - clears `additionalProperties` (reverting to the JSON Schema default of
//     "allowed"), so fields the SDK response struct doesn't model — for
//     example, new server-side fields — don't cause validation failures.
//
// The schema is mutated in place and returned for convenient chaining at the
// call site:
//
//	OutputSchema: helpers.WithOptionalFields(xxxListOutputSchema),
//
// Apply this only to list-tool schemas; single-entity `get_*` schemas should
// retain their strict `required` arrays so clients still receive useful
// constraints.
func WithOptionalFields(schema *jsonschema.Schema) *jsonschema.Schema {
	walkSchema(schema, func(s *jsonschema.Schema) {
		s.Required = nil
		s.AdditionalProperties = nil
	})
	return schema
}

// walkSchema invokes fn on s and every schema reachable from it via standard
// JSON Schema composition keywords. It is nil-safe.
func walkSchema(s *jsonschema.Schema, fn func(*jsonschema.Schema)) {
	if s == nil {
		return
	}
	fn(s)
	for _, p := range s.Properties {
		walkSchema(p, fn)
	}
	walkSchema(s.Items, fn)
	walkSchema(s.AdditionalProperties, fn)
	for _, b := range s.AnyOf {
		walkSchema(b, fn)
	}
	for _, b := range s.OneOf {
		walkSchema(b, fn)
	}
	for _, b := range s.AllOf {
		walkSchema(b, fn)
	}
}
