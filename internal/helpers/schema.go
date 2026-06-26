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
		Description: "If false, returns id + name only — useful when scanning many results.",
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
		Default: []byte(`true`),
	}
}

// MatchAllTagsSchema returns the schema for the boolean flag that switches
// tag filtering between AND (true) and OR (false) semantics.
func MatchAllTagsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "If true, match all tags; if false, match any.",
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
		Default: []byte(`false`),
	}
}

// UserGroupsSchema returns the schema for a user/team/company/job-role groups
// parameter. The object accepts user_ids, company_ids, team_ids, and/or
// job_role_ids arrays; at least one (and at most all four) must be supplied with
// non-empty values. When required is true the returned schema is a bare object;
// when false it is wrapped in AnyOf with null so the caller can omit the field.
// The caller supplies the purpose-specific framing as description (pass "" when
// the helper is used as a branch of an outer schema that already carries a
// description).
func UserGroupsSchema(description string, required bool) *jsonschema.Schema {
	obj := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"user_ids": {
				Type:     "array",
				Items:    &jsonschema.Schema{Type: "integer"},
				MinItems: new(1),
			},
			"company_ids": {
				Type:     "array",
				Items:    &jsonschema.Schema{Type: "integer"},
				MinItems: new(1),
			},
			"team_ids": {
				Type:     "array",
				Items:    &jsonschema.Schema{Type: "integer"},
				MinItems: new(1),
			},
			"job_role_ids": {
				Type:     "array",
				Items:    &jsonschema.Schema{Type: "integer"},
				MinItems: new(1),
			},
		},
		MinProperties: new(1),
		MaxProperties: new(4),
		AnyOf: []*jsonschema.Schema{
			{Required: []string{"user_ids"}},
			{Required: []string{"company_ids"}},
			{Required: []string{"team_ids"}},
			{Required: []string{"job_role_ids"}},
		},
	}
	if required {
		obj.Description = description
		return obj
	}
	return &jsonschema.Schema{
		Description: description,
		AnyOf: []*jsonschema.Schema{
			obj,
			{Type: "null"},
		},
	}
}

// DateTimeFilterSchema returns the schema for an optional RFC 3339 date-time
// filter parameter. The caller supplies the purpose-specific description.
func DateTimeFilterSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: description,
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Format: "date-time"},
			{Type: "null"},
		},
	}
}

// DateFilterSchema returns the schema for an optional ISO 8601 date
// (YYYY-MM-DD) filter parameter. The caller supplies the purpose-specific
// description.
func DateFilterSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: description,
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Format: "date"},
			{Type: "null"},
		},
	}
}
