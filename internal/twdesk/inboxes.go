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
	MethodInboxGet  toolsets.Method = "twdesk-get_inbox"
	MethodInboxList toolsets.Method = "twdesk-list_inboxes"
)

var (
	inboxListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodInboxGet)
	toolsets.RegisterMethod(MethodInboxList)

	var err error
	inboxListOutputSchema, err = jsonschema.For[deskmodels.InboxesResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for InboxListResponse: %v", err))
	}
}

// InboxGet finds a inbox in Teamwork Desk.  This will find it by ID
func InboxGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodInboxGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Inbox",
				ReadOnlyHint: true,
			},
			Description: `
				Retrieve detailed information about a specific inbox in Teamwork Desk by its ID
			`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the inbox to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			inbox, err := client.Inboxes.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get inbox: %w", err)
			}
			return helpers.NewToolResultText("Inbox retrieved successfully: %s", inbox.Inbox.Name), nil
		},
	}
}

// InboxList returns a list of inboxes that apply to the filters in Teamwork Desk
func InboxList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Type:        "array",
			Description: "The name of the inbox to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"email": {
			Type:        "array",
			Description: "The email of the inbox to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodInboxList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Inboxes",
				ReadOnlyHint: true,
			},
			Description: "List all inboxes in Teamwork Desk, with optional filters for name and email.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: inboxListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			// Apply filters to the inbox list
			name := arguments.GetStringSlice("name", []string{})
			email := arguments.GetStringSlice("email", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(email) > 0 {
				filter = filter.In("email", helpers.SliceToAny(email))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			inboxes, err := client.Inboxes.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list inboxes: %w", err)
			}
			return helpers.NewToolResultJSON(inboxes)
		},
	}
}
