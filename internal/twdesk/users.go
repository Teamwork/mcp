package twdesk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodUserGet  toolsets.Method = "twdesk-get_user"
	MethodUserList toolsets.Method = "twdesk-list_users"
)

func init() {
	toolsets.RegisterMethod(MethodUserGet)
	toolsets.RegisterMethod(MethodUserList)
}

// UserGet finds a user in Teamwork Desk.  This will find it by ID
func UserGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserGet),
			mcp.WithDescription(
				"Retrieve detailed information about a specific user in Teamwork Desk by their ID. "+
					"Useful for auditing user records, troubleshooting ticket assignments, or "+
					"integrating Desk user data into automation workflows."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the user to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			user, err := client.Users.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get user: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("User retrieved successfully: %s", user.User.FirstName)), nil
		},
	}
}

// UserList returns a list of users that apply to the filters in Teamwork Desk
func UserList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithOutputSchema[models.UsersResponse](),
		mcp.WithDescription(
			"List all users in Teamwork Desk, with optional filters for name, email, inbox, and part-time status. " +
				"Enables users to audit, analyze, or synchronize user configurations for support management, " +
				"reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("firstName", mcp.Description("The first names of the users to filter by.")),
		mcp.WithArray("lastName", mcp.Description("The last names of the users to filter by.")),
		mcp.WithArray("email", mcp.Description("The email addresses of the users to filter by.")),
		mcp.WithArray("inboxIDs", mcp.Description("The IDs of the inboxes to filter by.")),
		mcp.WithBoolean("isPartTime", mcp.Description("Whether to include part-time users in the results.")),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the user list
			firstNames := request.GetStringSlice("firstName", []string{})
			lastNames := request.GetStringSlice("lastName", []string{})
			emails := request.GetStringSlice("email", []string{})
			inboxIDs := request.GetIntSlice("inboxIDs", []int{})

			filter := deskclient.NewFilter()
			if len(firstNames) > 0 {
				filter = filter.In("firstName", helpers.SliceToAny(firstNames))
			}
			if len(lastNames) > 0 {
				filter = filter.In("lastName", helpers.SliceToAny(lastNames))
			}
			if len(emails) > 0 {
				filter = filter.In("email", helpers.SliceToAny(emails))
			}
			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", helpers.SliceToAny(inboxIDs))
			}

			isPartTime := request.GetBool("isPartTime", false)
			if isPartTime {
				filter = filter.Eq("isPartTime", true)
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, request)

			users, err := client.Users.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list users: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Users retrieved successfully: %v", users)), nil
		},
	}
}
