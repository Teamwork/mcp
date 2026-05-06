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
	properties["page"] = helpers.PageSchema()
	properties["pageSize"] = helpers.PageSizeSchema()
	properties["orderBy"] = helpers.OrderBySchema()
	properties["orderDirection"] = helpers.OrderDirectionSchema()
	return properties
}

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	v.Set("page", fmt.Sprintf("%d", arguments.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", arguments.GetInt("pageSize", 10)))
	v.Set("orderBy", arguments.GetString("orderBy", "createdAt"))
	v.Set("orderMode", arguments.GetString("orderDirection", "desc"))
}
