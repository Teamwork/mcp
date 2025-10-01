package twdesk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodInboxGet  toolsets.Method = "twdesk-get_inbox"
	MethodInboxList toolsets.Method = "twdesk-list_inboxes"
)

func init() {
	toolsets.RegisterMethod(MethodInboxGet)
	toolsets.RegisterMethod(MethodInboxList)
}

// InboxGet finds a inbox in Teamwork Desk.  This will find it by ID
func InboxGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodInboxGet),
			mcp.WithTitleAnnotation("Get Inbox"),
			mcp.WithDescription(`
				Retrieve detailed information about a specific inbox in Teamwork Desk by its ID
			`),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the inbox to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			inbox, err := client.Inboxes.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get inbox: %w", err)
			}

			return mcp.NewToolResultJSON(inbox)
		},
	}
}

// InboxList returns a list of inboxes that apply to the filters in Teamwork Desk
func InboxList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithTitleAnnotation("List Inboxes"),
		mcp.WithOutputSchema[deskmodels.InboxesResponse](),
		mcp.WithDescription(
			"List all inboxes in Teamwork Desk, with optional filters for name and email."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("name",
			mcp.Description("The name of the inbox to filter by."),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
		mcp.WithArray("email",
			mcp.Description("The email of the inbox to filter by."),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodInboxList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the inbox list
			name := request.GetStringSlice("name", []string{})
			email := request.GetStringSlice("email", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(email) > 0 {
				filter = filter.In("email", helpers.SliceToAny(email))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, request)

			inboxes, err := client.Inboxes.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list inboxes: %w", err)
			}

			return mcp.NewToolResultJSON(inboxes)
		},
	}
}
