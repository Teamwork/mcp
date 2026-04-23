package twspaces

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
)

// MethodSearch is the method name for searching pages in Teamwork Spaces.
const MethodSearch toolsets.Method = "twspaces-search"

// Search performs a full-text search across pages in Teamwork Spaces.
func Search(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSearch),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Search Spaces",
				ReadOnlyHint: true,
			},
			Description: "Perform a full-text search across pages in Teamwork Spaces. Supports filtering by " +
				"space, limiting results, and paginating through matches. Returns matching pages with " +
				"highlighted text snippets.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: paginationOptions(map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "The search query string.",
					},
					"spaceIds": {
						Type:        "array",
						Description: "Limit search to specific space IDs. Use 'twspaces-list_spaces' to find valid IDs.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"includeDeleted": {
						Type:        "boolean",
						Description: "Include deleted pages in search results.",
					},
				}),
				Required: []string{"query"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			filter := spacesmodels.SearchFilter{
				Query: arguments.GetString("query", ""),
			}

			if spaceIDs := arguments.GetIntSlice("spaceIds", nil); len(spaceIDs) > 0 {
				ids := make([]int64, len(spaceIDs))
				for i, id := range spaceIDs {
					ids[i] = int64(id)
				}
				filter.SpaceID = ids
			}

			if pageSize := arguments.GetInt("pageSize", 0); pageSize > 0 {
				l := int64(pageSize)
				filter.Limit = &l
			}

			if pageOffset := arguments.GetInt("pageOffset", 0); pageOffset > 0 {
				o := int64(pageOffset)
				filter.Offset = &o
			}

			filter.IncludeDel = arguments.GetBool("includeDeleted", false)

			results, err := client.Search.Search(ctx, filter)
			if err != nil {
				return nil, fmt.Errorf("failed to search: %w", err)
			}
			return helpers.NewToolResultJSON(results)
		},
	}
}
