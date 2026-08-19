package twdesk

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/pkg/helpers"
)

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intPtr returns a pointer to i, or nil if i is zero.
func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// falseSchema returns a schema that serialises to JSON boolean false.
// Used as AdditionalProperties: falseSchema() to satisfy OpenAI strict mode.
func falseSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Not: &jsonschema.Schema{}}
}

// paginationRequiredKeys returns the property names that paginationOptions
// injects, for use in strict-mode required lists.
func paginationRequiredKeys() []string {
	return []string{"page", "pageSize", "orderBy", "orderDirection", "fields"}
}

// Desk's page-size ceilings. These are lower than the v3 API's, which is why
// Desk does not use helpers.PageSizeSchema: advertising 500 to a caller whose
// request will come back holding 100 rows (or 50, on a search) reads as a short
// last page rather than as a cap, and nothing in the response says otherwise.
//
// Neither endpoint rejects an oversized page — both quietly rewrite it:
//   - list endpoints clamp anything over 100 to 100, and no query parameter
//     lifts that cap
//   - the search endpoints reset anything over 200, or under 1, to 50 — a
//     request for 250 comes back smaller than one for 200
const (
	maxPageSize       = 100.0
	maxSearchPageSize = 200.0
)

// pageSizeSchema returns the schema for the pageSize parameter of a Desk list
// endpoint.
func pageSizeSchema() *jsonschema.Schema {
	return deskPageSizeSchema(maxPageSize)
}

// searchPageSizeSchema returns the schema for the pageSize parameter of a Desk
// search endpoint, which allows a larger page than the list endpoints do.
func searchPageSizeSchema() *jsonschema.Schema {
	return deskPageSizeSchema(maxSearchPageSize)
}

func deskPageSizeSchema(maximum float64) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("Number of results per page for pagination (1-%[1]d). The API silently reduces "+
			"anything above %[1]d, so a larger value returns fewer results rather than more.", int(maximum)),
		AnyOf: []*jsonschema.Schema{
			{Type: "integer", Minimum: new(1.0), Maximum: new(maximum)},
			{Type: "null"},
		},
	}
}

// sparseFieldsSchema returns the JSON schema for the optional sparse fieldset parameter.
func sparseFieldsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Sparse fieldset: field names to include (e.g. [\"id\",\"name\"]). Omit to receive all fields.",
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
			{Type: "null"},
		},
	}
}

// getParams builds url.Values for a Get request with includes=all plus an
// optional sparse fieldset derived from the "fields" tool argument.
func getParams(arguments helpers.ToolArguments) url.Values {
	params := url.Values{}
	params.Set("includes", "all")

	fields := strings.Join(arguments.GetStringSlice("fields", nil), ",")
	if fields != "" {
		params.Set("fields", fields)
	}
	return params
}

func paginationOptions(properties map[string]*jsonschema.Schema) map[string]*jsonschema.Schema {
	return paginationOptionsWithPageSize(properties, pageSizeSchema())
}

// searchPaginationOptions is paginationOptions for the search endpoints, which
// take a larger page than the list endpoints. Only pageSize differs.
func searchPaginationOptions(properties map[string]*jsonschema.Schema) map[string]*jsonschema.Schema {
	return paginationOptionsWithPageSize(properties, searchPageSizeSchema())
}

func paginationOptionsWithPageSize(
	properties map[string]*jsonschema.Schema,
	pageSize *jsonschema.Schema,
) map[string]*jsonschema.Schema {
	if properties == nil {
		properties = make(map[string]*jsonschema.Schema)
	}
	properties["page"] = helpers.PageSchema()
	properties["pageSize"] = pageSize
	properties["orderBy"] = helpers.OrderBySchema()
	properties["orderDirection"] = helpers.OrderDirectionSchema()
	properties["fields"] = sparseFieldsSchema()
	return properties
}

// defaultPageSize is the page size used when the caller does not supply one.
const defaultPageSize = 10

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", defaultPageSize)))
	v.Set("orderBy", arguments.GetString("orderBy", "createdAt"))
	v.Set("orderMode", arguments.GetString("orderDirection", "desc"))

	fields := strings.Join(arguments.GetStringSlice("fields", nil), ",")
	if fields != "" {
		v.Set("fields", fields)
	}
}

// setSearchPagination applies pagination, ordering and the sparse fieldset to
// the query of a Desk *search* endpoint.
//
// It differs from setPagination in that ordering is only sent when the caller
// asked for it. The search endpoints document page and pageSize (see
// deskmodels.SearchHelpdocsFilter, and the pageSize/pageOffset the search
// response reports back), but no ordering parameter, so defaulting the sort
// would put an unverified parameter on every call. Forwarding it when asked
// costs nothing if the endpoint ignores it.
func setSearchPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", defaultPageSize)))

	orderBy := arguments.GetString("orderBy", "")
	if orderBy != "" {
		v.Set("orderBy", orderBy)
	}

	orderDirection := arguments.GetString("orderDirection", "")
	if orderDirection != "" {
		v.Set("orderMode", orderDirection)
	}

	fields := strings.Join(arguments.GetStringSlice("fields", nil), ",")
	if fields != "" {
		v.Set("fields", fields)
	}
}
