package twspaces

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
	properties["pageSize"] = helpers.PageSizeSchema()
	properties["pageOffset"] = helpers.PageOffsetSchema()
	return properties
}

func setPagination(v *url.Values, arguments helpers.ToolArguments) {
	if pageSize := arguments.GetInt("pageSize", 0); pageSize > 0 {
		v.Set("pageSize", fmt.Sprintf("%d", pageSize))
	}
	if pageOffset := arguments.GetInt("pageOffset", 0); pageOffset > 0 {
		v.Set("pageOffset", fmt.Sprintf("%d", pageOffset))
	}
}
