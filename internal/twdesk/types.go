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
	MethodTypeCreate toolsets.Method = "twdesk-create_type"
	MethodTypeUpdate toolsets.Method = "twdesk-update_type"
	MethodTypeGet    toolsets.Method = "twdesk-get_type"
	MethodTypeList   toolsets.Method = "twdesk-list_types"
)

var (
	typeGetOutputSchema  *jsonschema.Schema
	typeListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodTypeCreate)
	toolsets.RegisterMethod(MethodTypeUpdate)
	toolsets.RegisterMethod(MethodTypeGet)
	toolsets.RegisterMethod(MethodTypeList)

	var err error
	typeGetOutputSchema, err = jsonschema.For[deskmodels.TicketTypeResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TicketTypeResponse: %v", err))
	}

	typeListOutputSchema, err = jsonschema.For[deskmodels.TicketTypesResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TicketTypesResponse: %v", err))
	}
}

// TypeGet finds a type in Teamwork Desk.  This will find it by ID
func TypeGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Type",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific ticket type in Teamwork Desk by its ID. " +
				"Useful for auditing type usage, troubleshooting ticket categorization, or " +
				"integrating Desk type data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the type to retrieve.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: typeGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			t, err := client.TicketTypes.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get type: %w", err)
			}
			return helpers.NewToolResultJSON(t)
		},
	}
}

// TypeList returns a list of types that apply to the filters in Teamwork Desk
func TypeList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Type:        "array",
			Description: "The name of the type to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"inboxIDs": {
			Type:        "array",
			Description: "The inbox IDs of the type to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Types",
				ReadOnlyHint: true,
			},
			Description: "List all ticket types in Teamwork Desk, with optional filters for name and inbox association. " +
				"Enables users to audit, analyze, or synchronize type configurations for ticket management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: typeListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			// Apply filters to the type list
			name := arguments.GetStringSlice("name", []string{})
			inboxIDs := arguments.GetStringSlice("inboxIDs", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", helpers.SliceToAny(inboxIDs))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			types, err := client.TicketTypes.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list types: %w", err)
			}
			return helpers.NewToolResultJSON(types)
		},
	}
}

// TypeCreate creates a type in Teamwork Desk
func TypeCreate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Type",
			},
			Description: "Create a new ticket type in Teamwork Desk by specifying its name, display order, and future " +
				"inbox settings. Useful for customizing ticket workflows, introducing new categories, or " +
				"adapting Desk to evolving support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the type.",
					},
					"displayOrder": {
						Type:        "integer",
						Description: "The display order of the type.",
					},
					"enabledForFutureInboxes": {
						Type:        "boolean",
						Description: "Whether the type is enabled for future inboxes.",
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

			t, err := client.TicketTypes.Create(ctx, &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    arguments.GetString("name", ""),
					DisplayOrder:            arguments.GetInt("displayOrder", 0),
					EnabledForFutureInboxes: arguments.GetBool("enabledForFutureInboxes", false),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create type: %w", err)
			}
			return helpers.NewToolResultText("Type created successfully with ID %d", t.TicketType.ID), nil
		},
	}
}

// TypeUpdate updates a type in Teamwork Desk
func TypeUpdate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Type",
			},
			Description: "Update an existing ticket type in Teamwork Desk by ID, allowing changes to its name, " +
				"display order, and future inbox settings. Supports evolving support policies, rebranding, or correcting " +
				"type attributes for improved ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the type to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the type.",
					},
					"displayOrder": {
						Type:        "integer",
						Description: "The display order of the type.",
					},
					"enabledForFutureInboxes": {
						Type:        "boolean",
						Description: "Whether the type is enabled for future inboxes.",
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

			_, err = client.TicketTypes.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    arguments.GetString("name", ""),
					DisplayOrder:            arguments.GetInt("displayOrder", 0),
					EnabledForFutureInboxes: arguments.GetBool("enabledForFutureInboxes", false),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create type: %w", err)
			}

			return helpers.NewToolResultText("Type updated successfully"), nil
		},
	}
}
