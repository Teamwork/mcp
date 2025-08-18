package twdesk

import (
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

func paginationOptions() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithNumber("page", mcp.Description("The page number to retrieve.")),
		mcp.WithNumber("pageSize", mcp.Description("The number of results to retrieve per page.")),
		mcp.WithString("orderBy", mcp.Description("The field to order the results by.")),
		mcp.WithString("orderDirection", mcp.Description("The direction to order the results by (asc, desc).")),
	}
}

func setPagination(v *url.Values, request mcp.CallToolRequest) {
	v.Set("page", fmt.Sprintf("%d", request.GetInt("page", 1)))
	v.Set("pageSize", fmt.Sprintf("%d", request.GetInt("pageSize", 10)))
	v.Set("orderBy", request.GetString("orderBy", "createdAt"))
	v.Set("orderMode", request.GetString("orderDirection", "desc"))
}
