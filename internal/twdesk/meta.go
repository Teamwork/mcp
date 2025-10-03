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
		Type:        "integer",
		Description: "The page number to retrieve.",
	}
	properties["pageSize"] = &jsonschema.Schema{
		Type:        "integer",
		Description: "The number of results to retrieve per page.",
	}
	properties["orderBy"] = &jsonschema.Schema{
		Type:        "string",
		Description: "The field to order the results by.",
	}
	properties["orderDirection"] = &jsonschema.Schema{
		Type:        "string",
		Description: "The direction to order the results by (asc, desc).",
	}
	return properties
}

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", 10)))
	v.Set("orderBy", arguments.GetString("orderBy", "createdAt"))
	v.Set("orderMode", arguments.GetString("orderDirection", "desc"))
}
