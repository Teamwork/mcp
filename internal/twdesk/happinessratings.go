package twdesk

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodHappinessRatingOptionList toolsets.Method = "twdesk-list_happiness_rating_options"
)

// happinessRatingOptionService addresses the happiness rating option routes,
// which the SDK models no service for.
func happinessRatingOptionService(client *deskclient.Client) *deskclient.Service[map[string]any, map[string]any] {
	return deskclient.NewService[map[string]any, map[string]any](client,
		deskclient.NewDefaultPathHandler("happinessratingoptions"),
	)
}

// HappinessRatingOptionList lists the ratings a customer can give a ticket in
// Teamwork Desk.
func HappinessRatingOptionList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHappinessRatingOptionList),
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Happiness Rating Options",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "List the happiness ratings a customer can give a ticket, each with its name and " +
				"score. The IDs are what happinessRatingIDs on twdesk-search_tickets takes; they are " +
				"specific to the installation, so do not pass a score in their place.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           paginationOptions(nil),
				Required:             paginationRequiredKeys(),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			params := url.Values{}
			setPagination(&params, arguments)

			options, err := happinessRatingOptionService(client).List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list happiness rating options")
			}
			return helpers.NewToolResultJSON(*options)
		},
	}
}
