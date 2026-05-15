package helpers

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// PageSchema returns the schema for a page-number pagination parameter.
func PageSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Page number for pagination of results.",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "null"},
		},
	}
}

// PageSizeSchema returns the schema for a page-size pagination parameter.
func PageSizeSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Number of results per page for pagination.",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "null"},
		},
	}
}

// PageOffsetSchema returns the schema for an offset-based pagination parameter
// (used by APIs that take a starting index rather than a page number).
func PageOffsetSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "The index position to start retrieving results from (not a page number).",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "null"},
		},
	}
}

// OrderBySchema returns the schema for an order-by parameter.
func OrderBySchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "The field to order the results by.",
		AnyOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}
}

// OrderDirectionSchema returns the schema for an order-direction parameter
// accepting "asc" or "desc".
func OrderDirectionSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "The direction to order the results by (asc, desc).",
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Enum: []any{"asc", "desc"}},
			{Type: "null"},
		},
	}
}

// SearchTermSchema returns the schema for a search-term filter parameter.
// fields describes what is searched (e.g. "name", "name or description").
func SearchTermSchema(entity, fields string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("A search term to filter %s by %s.", entity, fields),
		AnyOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}
}

// TagIDsFilterSchema returns the schema for a tag-IDs list used to filter
// listings by tag.
func TagIDsFilterSchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("A list of tag IDs to filter %s by tags.", entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			{Type: "null"},
		},
	}
}

// TagIDsAssociateSchema returns the schema for a tag-IDs list used to attach
// tags when creating or updating an entity.
func TagIDsAssociateSchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("A list of tag IDs to associate with the %s.", entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			{Type: "null"},
		},
	}
}

// VerboseSchema returns the schema for a verbose flag controlling response
// detail level. When true (default), full entity details are returned; when
// false, sparse fieldsets are applied to reduce response size. Structured
// content is always returned; list-tool output schemas are relaxed (all
// fields optional) so sparse payloads still validate.
func VerboseSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "If true (default), the response includes full entity details. " +
			"If false, only a minimal subset of fields (typically an id and a name/title) is returned to " +
			"reduce response size — useful when scanning many results to pick an id before fetching the full entity.",
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
		Default: []byte(`true`),
	}
}

// MatchAllTagsSchema returns the schema for the boolean flag that switches
// tag filtering between AND (true) and OR (false) semantics.
func MatchAllTagsSchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf(
			"If true, the search will match %s that have all the specified tags. "+
				"If false, the search will match %s that have any of the specified tags. "+
				"Defaults to false.",
			entity, entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
	}
}
