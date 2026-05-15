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
	MethodTypeCreate toolsets.Method = "twdesk-create_ticket_type"
	MethodTypeUpdate toolsets.Method = "twdesk-update_ticket_type"
	MethodTypeGet    toolsets.Method = "twdesk-get_ticket_type"
	MethodTypeList   toolsets.Method = "twdesk-list_ticket_types"
)

// TypeGet finds a type in Teamwork Desk.  This will find it by ID
func TypeGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Ticket Type",
				ReadOnlyHint: true,
			},
			Description: "Get ticket type.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket type to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id", "fields"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			t, err := client.TicketTypes.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return nil, fmt.Errorf("failed to get type: %w", err)
			}
			return helpers.NewToolResultJSON(t)
		},
	}
}

// TypeList returns a list of types that apply to the filters in Teamwork Desk
func TypeList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the type to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"inboxIDs": {
			Description: "The IDs of the inboxes to filter by. Inbox IDs can be found by using the 'twdesk-list_inboxes' tool.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Ticket Types",
				ReadOnlyHint: true,
			},
			Description: "List ticket types. Filter by name or inbox.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required:             append(paginationRequiredKeys(), "name", "inboxIDs"),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Apply filters to the type list
			name := arguments.GetStringSlice("name", []string{})
			inboxIDs := arguments.GetIntSlice("inboxIDs", []int{})

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
func TypeCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Ticket Type",
			},
			Description: "Create ticket type.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the ticket type.",
					},
					"displayOrder": {
						Description: "The display order of the type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"enabledForFutureInboxes": {
						Description: "Whether the type is enabled for future inboxes.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name", "displayOrder", "enabledForFutureInboxes"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			name := arguments.GetString("name", "")
			enabledForFutureInboxes := arguments.GetBool("enabledForFutureInboxes", false)
			t, err := client.TicketTypes.Create(ctx, &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    &name,
					DisplayOrder:            intPtr(arguments.GetInt("displayOrder", 0)),
					EnabledForFutureInboxes: boolPtr(enabledForFutureInboxes),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create type: %w", err)
			}
			return helpers.NewToolResultText("Ticket type created successfully with ID %d", t.TicketType.ID), nil
		},
	}
}

// TypeUpdate updates a type in Teamwork Desk
func TypeUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTypeUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Ticket Type",
			},
			Description: "Update ticket type.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket type to update.",
					},
					"name": {
						Description: "The new name of the type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"displayOrder": {
						Description: "The display order of the type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"enabledForFutureInboxes": {
						Description: "Whether the type is enabled for future inboxes.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id", "name", "displayOrder", "enabledForFutureInboxes"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			enabledForFutureInboxes := arguments.GetBool("enabledForFutureInboxes", false)
			_, err = client.TicketTypes.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    strPtr(arguments.GetString("name", "")),
					DisplayOrder:            intPtr(arguments.GetInt("displayOrder", 0)),
					EnabledForFutureInboxes: boolPtr(enabledForFutureInboxes),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update ticket type: %w", err)
			}

			return helpers.NewToolResultText("Ticket type updated successfully"), nil
		},
	}
}
