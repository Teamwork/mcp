package twdesk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	MethodUserGet  toolsets.Method = "twdesk-get_user"
	MethodUserList toolsets.Method = "twdesk-list_users"
)

var (
	userGetOutputSchema  *jsonschema.Schema
	userListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodUserGet)
	toolsets.RegisterMethod(MethodUserList)

	var err error
	userGetOutputSchema, err = jsonschema.For[deskmodels.UserResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserResponse: %v", err))
	}

	userListOutputSchema, err = jsonschema.For[deskmodels.UsersResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UsersResponse: %v", err))
	}
}

// UserGet finds a user in Teamwork Desk.  This will find it by ID
func UserGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodUserGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get User",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific user in Teamwork Desk by their ID. " +
				"Useful for auditing user records, troubleshooting ticket assignments, or " +
				"integrating Desk user data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to retrieve.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: userGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			user, err := client.Users.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get user: %w", err)
			}

			return helpers.NewToolResultText(fmt.Sprintf("User retrieved successfully: %s", user.User.FirstName)), nil
		},
	}
}

// UserList returns a list of users that apply to the filters in Teamwork Desk
func UserList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"firstName": {
			Type:        "array",
			Description: "The first names of the users to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"lastName": {
			Type:        "array",
			Description: "The last names of the users to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"email": {
			Type:        "array",
			Description: "The email addresses of the users to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"inboxIDs": {
			Type:        "array",
			Description: "The IDs of the inboxes to filter by.",
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"isPartTime": {
			Type:        "boolean",
			Description: "Whether to include part-time users in the results.",
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodUserList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Users",
				ReadOnlyHint: true,
			},
			Description: "List all users in Teamwork Desk, with optional filters for name, email, inbox, and part-time status. " +
				"Enables users to audit, analyze, or synchronize user configurations for support management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: userListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			// Apply filters to the user list
			firstNames := arguments.GetStringSlice("firstName", []string{})
			lastNames := arguments.GetStringSlice("lastName", []string{})
			emails := arguments.GetStringSlice("email", []string{})
			inboxIDs := arguments.GetIntSlice("inboxIDs", []int{})

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

			isPartTime := arguments.GetBool("isPartTime", false)
			if isPartTime {
				filter = filter.Eq("isPartTime", true)
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			users, err := client.Users.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list users: %w", err)
			}

			return helpers.NewToolResultText(fmt.Sprintf("Users retrieved successfully: %v", users)), nil
		},
	}
}
