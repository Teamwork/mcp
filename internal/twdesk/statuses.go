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
	MethodStatusCreate toolsets.Method = "twdesk-create_status"
	MethodStatusUpdate toolsets.Method = "twdesk-update_status"
	MethodStatusGet    toolsets.Method = "twdesk-get_status"
	MethodStatusList   toolsets.Method = "twdesk-list_statuses"
)

var (
	statusListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodStatusCreate)
	toolsets.RegisterMethod(MethodStatusUpdate)
	toolsets.RegisterMethod(MethodStatusGet)
	toolsets.RegisterMethod(MethodStatusList)

	var err error
	statusListOutputSchema, err = jsonschema.For[deskmodels.TicketStatusesResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for StatusListResponse: %v", err))
	}
}

// StatusGet finds a status in Teamwork Desk.  This will find it by ID
func StatusGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Status",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific status in Teamwork Desk by its ID. " +
				"Useful for auditing status usage, troubleshooting ticket workflows, or " +
				"integrating Desk status data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the status to retrieve.",
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

			status, err := client.TicketStatuses.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}

			return helpers.NewToolResultText("Status retrieved successfully: %s", status.TicketStatus.Name), nil
		},
	}
}

// StatusList returns a list of statuses that apply to the filters in Teamwork Desk
func StatusList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Type:        "array",
			Description: "The name of the status to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"color": {
			Type:        "array",
			Description: "The color of the status to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"code": {
			Type:        "array",
			Description: "The code of the status to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Statuses",
				ReadOnlyHint: true,
			},
			Description: "List all statuses in Teamwork Desk, with optional filters for name, color, and code. " +
				"Enables users to audit, analyze, or synchronize status configurations for ticket management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: statusListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			// Apply filters to the status list
			name := arguments.GetStringSlice("name", []string{})
			color := arguments.GetStringSlice("color", []string{})
			code := arguments.GetStringSlice("code", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(color) > 0 {
				filter = filter.In("color", helpers.SliceToAny(color))
			}
			if len(code) > 0 {
				filter = filter.In("code", helpers.SliceToAny(code))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			statuses, err := client.TicketStatuses.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list statuses: %w", err)
			}
			return helpers.NewToolResultJSON(statuses)
		},
	}
}

// StatusCreate creates a status in Teamwork Desk
func StatusCreate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Status",
			},
			Description: "Create a new status in Teamwork Desk by specifying its name, color, and display order. " +
				"Useful for customizing ticket workflows, introducing new resolution states, or " +
				"adapting Desk to evolving support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the status.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the status.",
					},
					"displayOrder": {
						Type:        "integer",
						Description: "The display order of the status.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			status, err := client.TicketStatuses.Create(ctx, &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         arguments.GetString("name", ""),
					Color:        arguments.GetString("color", ""),
					DisplayOrder: arguments.GetInt("displayOrder", 0),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create status: %w", err)
			}
			return helpers.NewToolResultText("Status created successfully with ID %d", status.TicketStatus.ID), nil
		},
	}
}

// StatusUpdate updates a status in Teamwork Desk
func StatusUpdate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Status",
			},
			Description: "Update an existing status in Teamwork Desk by ID, allowing changes to its name, color, and " +
				"display order. Supports evolving support policies, rebranding, or correcting status attributes for improved " +
				"ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the status to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the status.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the status.",
					},
					"displayOrder": {
						Type:        "integer",
						Description: "The display order of the status.",
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

			_, err = client.TicketStatuses.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         arguments.GetString("name", ""),
					Color:        arguments.GetString("color", ""),
					DisplayOrder: arguments.GetInt("displayOrder", 0),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create status: %w", err)
			}

			return helpers.NewToolResultText("Status updated successfully"), nil
		},
	}
}
