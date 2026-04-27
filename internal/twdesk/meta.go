package twdesk

import (
	"fmt"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/helpers"
)

func paginationOptions(properties map[string]*jsonschema.Schema) map[string]*jsonschema.Schema {
	if properties == nil {
		properties = make(map[string]*jsonschema.Schema)
	}
	properties["page"] = &jsonschema.Schema{
		Description: "The page number to retrieve.",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "null"},
		},
	}
	properties["pageSize"] = &jsonschema.Schema{
		Description: "The number of results to retrieve per page.",
		AnyOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "null"},
		},
	}
	properties["orderBy"] = &jsonschema.Schema{
		Description: "The field to order the results by.",
		AnyOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}
	properties["orderDirection"] = &jsonschema.Schema{
		Description: "The direction to order the results by (asc, desc).",
		AnyOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}
	return properties
}

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", 10)))
	v.Set("orderBy", arguments.GetString("orderBy", "createdAt"))
	v.Set("orderMode", arguments.GetString("orderDirection", "desc"))
}
