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
	MethodSourceList toolsets.Method = "twdesk-list_ticket_sources"
)

// ticketSourceService addresses the ticket source routes. The SDK's own
// TicketSourceService decodes into a typed struct; this one hands the body back
// untouched, like the other lookup tools, so no attribute is lost on the way.
func ticketSourceService(client *deskclient.Client) *deskclient.Service[map[string]any, map[string]any] {
	return deskclient.NewService[map[string]any, map[string]any](client,
		deskclient.NewDefaultPathHandler("ticketsources"),
	)
}

// SourceList lists the channels a ticket can arrive through in Teamwork Desk.
func SourceList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSourceList),
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Ticket Sources",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "List ticket sources: the channels a ticket can arrive through, such as email, " +
				"a contact form or the API. The IDs are what sourceIDs on twdesk-search_tickets takes.",
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

			sources, err := ticketSourceService(client).List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list ticket sources")
			}
			return helpers.NewToolResultJSON(*sources)
		},
	}
}
