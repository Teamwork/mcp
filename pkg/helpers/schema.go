package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// maxPageSize is the largest page the v3 API accepts
const maxPageSize = 500.0

// PageSchema returns the schema for a page-number pagination parameter.
func PageSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Page number for pagination of results (1-based).",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer", Minimum: new(1.0)},
			{Type: "null"},
		},
	}
}

// PageSizeSchema returns the schema for a page-size pagination parameter.
//
// The bounds are the ones the v3 API enforces
// (https://apidocs.teamwork.com/guides/teamwork/how-does-paging-work): it
// rejects anything above 500 with a 400, so declaring the ceiling turns a
// wasted round trip into a client-side validation error.
func PageSizeSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Number of results per page for pagination (1-500).",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer", Minimum: new(1.0), Maximum: new(maxPageSize)},
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

// FieldsSchema returns the schema for the sparse-fieldset parameter of a list
// or get tool. E is the SDK entity struct the tool selects attributes of, the
// same one its OptionalFieldsParam validates against:
//
//	helpers.FieldsSchema[projects.Task]("task")
//
// The accepted names are enumerated off E, so the enum cannot drift from the
// validator. The enum is the only place a model can read them: tool definitions
// carry no output schema, so without it the vocabulary is learnt one rejection
// at a time.
func FieldsSchema[E any](entity string) *jsonschema.Schema {
	names := SparseFieldNames[string, E]()
	// Nil for a non-struct E: a missing constraint, not one that rejects
	// everything.
	var enum []any
	for _, name := range names {
		enum = append(enum, name)
	}
	return &jsonschema.Schema{
		Description: fmt.Sprintf("The attributes to return for each %s, from the listed names.", entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "string", Enum: enum}},
			{Type: "null"},
		},
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

// NotifySchema returns the schema for a "notify" parameter. Every shape must
// be advertised here: the MCP SDK validates arguments against the schema
// before the handler runs, so parseNotify (twprojects) can only coerce what
// the schema allows. false = notify nobody. true = followers and the default
// when withFollowers (comments), otherwise an alias for "all", the default.
func NotifySchema(description string, withFollowers bool) *jsonschema.Schema {
	defaultValue := json.RawMessage(`"all"`)
	boolDescription := `true is the same as "all": notify all project members. false notifies nobody.`
	boolPhrase := `, the boolean true as an alias for "all"`
	if withFollowers {
		defaultValue = json.RawMessage(`true`)
		boolDescription = "true notifies all followers of the entity this comment is related to. " +
			"false notifies nobody."
		boolPhrase = ", the boolean true to notify all followers of the related entity"
	}
	description += ` Accepts the string "all" to notify all project members` + boolPhrase +
		`, the boolean false to notify nobody, a plain array of user IDs (e.g. [123, 456]), ` +
		`or an object selecting user_ids, company_ids, team_ids and/or job_role_ids ` +
		`(e.g. {"user_ids": [123, 456]}).`
	return &jsonschema.Schema{
		Description: description,
		Default:     defaultValue,
		AnyOf: []*jsonschema.Schema{
			{
				Type:        "string",
				Description: "Notify all project members.",
				Enum:        []any{"all"},
			},
			{
				Type:        "boolean",
				Description: boolDescription,
			},
			{
				Type:        "array",
				Description: `The IDs of the users to notify; shorthand for {"user_ids": [...]}.`,
				Items:       &jsonschema.Schema{Type: "integer"},
				MinItems:    new(1),
			},
			UserGroupsSchema("", true),
			{Type: "null"},
		},
	}
}

// DateTimeFilterSchema returns the schema for an optional date-time filter
// parameter. The caller supplies the purpose-specific description.
//
// Both accepted forms are spelled out with examples because a model asked about
// a date range emits a bare YYYY-MM-DD by default: a schema that advertises only
// format "date-time" costs a failed first call and a visible retry. The binders
// accept the plain date (see dateTimeLayouts), and the description says what it
// means so the caller is not left guessing whether the end of the range is
// included.
func DateTimeFilterSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: description + " Accepts an RFC 3339 timestamp (2026-08-03T14:30:00Z) or a plain " +
			"YYYY-MM-DD date (2026-08-03), which covers that whole day in UTC.",
		Examples: []any{"2026-08-03", "2026-08-03T14:30:00Z"},
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Format: "date-time"},
			{Type: "string", Format: "date"},
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
