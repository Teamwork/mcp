package twdesk

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/helpers"
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

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool {
	return &b
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
	if fields := strings.Join(arguments.GetStringSlice("fields", nil), ","); fields != "" {
		params.Set("fields", fields)
	}
	return params
}

func paginationOptions(properties map[string]*jsonschema.Schema) map[string]*jsonschema.Schema {
	if properties == nil {
		properties = make(map[string]*jsonschema.Schema)
	}
	properties["page"] = helpers.PageSchema()
	properties["pageSize"] = helpers.PageSizeSchema()
	properties["orderBy"] = helpers.OrderBySchema()
	properties["orderDirection"] = helpers.OrderDirectionSchema()
	properties["fields"] = sparseFieldsSchema()
	return properties
}

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", 10)))
	v.Set("orderBy", arguments.GetString("orderBy", "createdAt"))
	v.Set("orderMode", arguments.GetString("orderDirection", "desc"))
	if fields := strings.Join(arguments.GetStringSlice("fields", nil), ","); fields != "" {
		v.Set("fields", fields)
	}
}
