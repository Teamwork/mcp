package helpers

import (
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

// metaWebLinkSchema returns the schema definition for a meta object containing
// a webLink field. This schema describes the meta section injected by
// WebLinker at runtime into API responses.
func metaWebLinkSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"webLink": {
				Type:        "string",
				Description: "A direct URL to this resource in the Teamwork.com web application.",
			},
		},
	}
}

// WithMetaWebLinkSchema patches a JSON schema generated from an API response
// type to include a meta.webLink property on entity objects. This aligns the
// output schema with the runtime behavior of WebLinker, which injects web links
// into serialized responses.
//
// If the top-level schema is itself an array of objects, the "meta" property
// with a "webLink" field is added to each item schema.
//
// Otherwise, the function walks the top-level properties of the schema and, for
// each property that is not a known root field (like "meta" or "included"):
//   - If the property is an object, it adds a "meta" property with a "webLink"
//     field to that object's schema.
//   - If the property is an array of objects, it adds the "meta" property to
//     the array item schema.
//
// If the entity object already has a "meta" property in its schema, the
// "webLink" field is merged into the existing meta schema without overwriting
// any existing "webLink" definition.
//
// The schema is modified in place and also returned for convenient chaining:
//
//	schema, err = jsonschema.For[Response](opts)
//	schema = helpers.WithMetaWebLinkSchema(schema)
func WithMetaWebLinkSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return schema
	}

	for key, prop := range schema.Properties {
		if slices.Contains(knownRootFields, key) || prop == nil {
			continue
		}

		switch {
		case prop.Type == "object" && prop.Properties != nil:
			// Single entity: {"task": {"id": 123, ...}}
			addMetaWebLinkToSchema(prop)

		case isSchemaArray(prop) && prop.Items != nil && prop.Items.Properties != nil:
			// Array of entities: {"tasks": [{"id": 123, ...}, ...]}
			addMetaWebLinkToSchema(prop.Items)
		}
	}

	return schema
}

// isSchemaArray reports whether a schema represents an array type, handling
// both single type ("array") and union types (["null", "array"]).
func isSchemaArray(s *jsonschema.Schema) bool {
	return s.Type == "array" || slices.Contains(s.Types, "array")
}

// addMetaWebLinkToSchema adds or merges a meta.webLink property into an object
// schema. If the schema already has a "meta" property, the "webLink" field is
// added to the existing meta properties (unless it already exists). Otherwise,
// a new "meta" property is created with the webLink definition.
func addMetaWebLinkToSchema(objectSchema *jsonschema.Schema) {
	if objectSchema.Properties == nil {
		objectSchema.Properties = make(map[string]*jsonschema.Schema)
	}

	webLinkSchema := &jsonschema.Schema{
		Type:        "string",
		Description: "A direct URL to this resource in the Teamwork.com web application.",
	}

	if existing, ok := objectSchema.Properties["meta"]; ok && existing != nil {
		// Meta property already exists; merge webLink into it.
		if existing.Properties == nil {
			existing.Properties = make(map[string]*jsonschema.Schema)
		}
		if _, hasWebLink := existing.Properties["webLink"]; !hasWebLink {
			existing.Properties["webLink"] = webLinkSchema
		}
	} else {
		objectSchema.Properties["meta"] = metaWebLinkSchema()
	}
}
