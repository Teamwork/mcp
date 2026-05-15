package twdesk

import (
	"context"
	"fmt"
	"net/http"
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
	MethodPriorityCreate toolsets.Method = "twdesk-create_priority"
	MethodPriorityUpdate toolsets.Method = "twdesk-update_priority"
	MethodPriorityGet    toolsets.Method = "twdesk-get_priority"
	MethodPriorityList   toolsets.Method = "twdesk-list_priorities"
)

// PriorityGet finds a priority in Teamwork Desk.  This will find it by ID
func PriorityGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Priority",
				ReadOnlyHint: true,
			},
			Description: "Get ticket priority.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the priority to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			priority, err := client.TicketPriorities.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return nil, fmt.Errorf("failed to get priority: %w", err)
			}
			return helpers.NewToolResultJSON(priority)
		},
	}
}

// PriorityList returns a list of priorities that apply to the filters in Teamwork Desk
func PriorityList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the priority to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"color": {
			Description: "The color of the priority to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Priorities",
				ReadOnlyHint: true,
			},
			Description: "List ticket priorities. Filter by name or color.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required:   []string{},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Apply filters to the priority list
			name := arguments.GetStringSlice("name", []string{})
			color := arguments.GetStringSlice("color", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(color) > 0 {
				filter = filter.In("color", helpers.SliceToAny(color))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			priorities, err := client.TicketPriorities.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list priorities: %w", err)
			}
			return helpers.NewToolResultJSON(priorities)
		},
	}
}

// PriorityCreate creates a priority in Teamwork Desk
func PriorityCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Priority",
			},
			Description: "Create ticket priority.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the priority.",
					},
					"color": {
						Description: "The color of the priority.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			name := arguments.GetString("name", "")
			priority, err := client.TicketPriorities.Create(ctx, &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  &name,
					Color: strPtr(arguments.GetString("color", "")),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create priority: %w", err)
			}
			return helpers.NewToolResultText("Priority created successfully with ID %d", priority.TicketPriority.ID), nil
		},
	}
}

// PriorityUpdate updates a priority in Teamwork Desk
func PriorityUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Priority",
			},
			Description: "Update ticket priority.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the priority to update.",
					},
					"name": {
						Description: "The new name of the priority.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"color": {
						Description: "The color of the priority.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			_, err = client.TicketPriorities.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  strPtr(arguments.GetString("name", "")),
					Color: strPtr(arguments.GetString("color", "")),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update priority: %w", err)
			}

			return helpers.NewToolResultText("Priority updated successfully"), nil
		},
	}
}
