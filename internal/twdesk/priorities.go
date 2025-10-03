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
	MethodPriorityCreate toolsets.Method = "twdesk-create_priority"
	MethodPriorityUpdate toolsets.Method = "twdesk-update_priority"
	MethodPriorityGet    toolsets.Method = "twdesk-get_priority"
	MethodPriorityList   toolsets.Method = "twdesk-list_priorities"
)

var (
	priorityGetOutputSchema  *jsonschema.Schema
	priorityListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodPriorityCreate)
	toolsets.RegisterMethod(MethodPriorityUpdate)
	toolsets.RegisterMethod(MethodPriorityGet)
	toolsets.RegisterMethod(MethodPriorityList)

	var err error
	priorityGetOutputSchema, err = jsonschema.For[deskmodels.TicketPriorityResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for PriorityGetResponse: %v", err))
	}

	priorityListOutputSchema, err = jsonschema.For[deskmodels.TicketPrioritiesResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for PriorityListResponse: %v", err))
	}
}

// PriorityGet finds a priority in Teamwork Desk.  This will find it by ID
func PriorityGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Priority",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific priority in Teamwork Desk by its ID. " +
				"Useful for inspecting priority attributes, troubleshooting ticket routing, or " +
				"integrating Desk priority data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the priority to retrieve.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: priorityGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			priority, err := client.TicketPriorities.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get priority: %w", err)
			}

			name := priority.TicketPriority.Name
			return helpers.NewToolResultText(fmt.Sprintf("Priority retrieved successfully: %s", name)), nil
		},
	}
}

// PriorityList returns a list of priorities that apply to the filters in Teamwork Desk
func PriorityList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Type:        "array",
			Description: "The name of the priority to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"color": {
			Type:        "array",
			Description: "The color of the priority to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
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
			Description: "List all available priorities in Teamwork Desk, with optional filters for name and color. " +
				"Enables users to audit, analyze, or synchronize priority configurations for ticket management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: priorityListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
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

			return helpers.NewToolResultText(fmt.Sprintf("Priorities retrieved successfully: %v", priorities)), nil
		},
	}
}

// PriorityCreate creates a priority in Teamwork Desk
func PriorityCreate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Priority",
			},
			Description: "Create a new priority in Teamwork Desk by specifying its name and color. Useful for customizing " +
				"ticket workflows, introducing new escalation levels, or adapting Desk to evolving support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the priority.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the priority.",
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

			priority, err := client.TicketPriorities.Create(ctx, &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  arguments.GetString("name", ""),
					Color: arguments.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create priority: %w", err)
			}

			id := priority.TicketPriority.ID
			return helpers.NewToolResultText(fmt.Sprintf("Priority created successfully with ID %d", id)), nil
		},
	}
}

// PriorityUpdate updates a priority in Teamwork Desk
func PriorityUpdate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPriorityUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Priority",
			},
			Description: "Update an existing priority in Teamwork Desk by ID, allowing changes to its name and color. " +
				"Supports evolving support policies, rebranding, or correcting priority attributes for improved " +
				"ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the priority to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the priority.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the priority.",
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

			_, err = client.TicketPriorities.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  arguments.GetString("name", ""),
					Color: arguments.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create priority: %w", err)
			}

			return helpers.NewToolResultText("Priority updated successfully"), nil
		},
	}
}
