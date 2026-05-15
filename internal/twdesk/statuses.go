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
	MethodStatusCreate toolsets.Method = "twdesk-create_status"
	MethodStatusUpdate toolsets.Method = "twdesk-update_status"
	MethodStatusGet    toolsets.Method = "twdesk-get_status"
	MethodStatusList   toolsets.Method = "twdesk-list_statuses"
)

// StatusGet finds a status in Teamwork Desk.  This will find it by ID
func StatusGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Status",
				ReadOnlyHint: true,
			},
			Description: "Get ticket status.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the status to retrieve.",
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

			status, err := client.TicketStatuses.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}

			return helpers.NewToolResultJSON(status)
		},
	}
}

// StatusList returns a list of statuses that apply to the filters in Teamwork Desk
func StatusList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the status to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"color": {
			Description: "The color of the status to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"code": {
			Description: "The code of the status to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
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
			Description: "List ticket statuses. Filter by name, color, or code.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required:             append(paginationRequiredKeys(), "name", "color", "code"),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
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
func StatusCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Status",
			},
			Description: "Create ticket status.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the status.",
					},
					"color": {
						Description: "The color of the status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"displayOrder": {
						Description: "The display order of the status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name", "color", "displayOrder"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			name := arguments.GetString("name", "")
			status, err := client.TicketStatuses.Create(ctx, &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         &name,
					Color:        strPtr(arguments.GetString("color", "")),
					DisplayOrder: intPtr(arguments.GetInt("displayOrder", 0)),
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
func StatusUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodStatusUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Status",
			},
			Description: "Update ticket status.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the status to update.",
					},
					"name": {
						Description: "The new name of the status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"color": {
						Description: "The color of the status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"displayOrder": {
						Description: "The display order of the status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id", "name", "color", "displayOrder"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			_, err = client.TicketStatuses.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         strPtr(arguments.GetString("name", "")),
					Color:        strPtr(arguments.GetString("color", "")),
					DisplayOrder: intPtr(arguments.GetInt("displayOrder", 0)),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update status: %w", err)
			}

			return helpers.NewToolResultText("Status updated successfully"), nil
		},
	}
}
