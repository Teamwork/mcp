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
	MethodStatusCreate toolsets.Method = "twdesk-create_status"
	MethodStatusUpdate toolsets.Method = "twdesk-update_status"
	MethodStatusGet    toolsets.Method = "twdesk-get_status"
	MethodStatusList   toolsets.Method = "twdesk-list_statuses"
)

func init() {
	toolsets.RegisterMethod(MethodStatusCreate)
	toolsets.RegisterMethod(MethodStatusUpdate)
	toolsets.RegisterMethod(MethodStatusGet)
	toolsets.RegisterMethod(MethodStatusList)
}

// StatusGet finds a status in Teamwork Desk.  This will find it by ID
func StatusGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodStatusGet),
			mcp.WithTitleAnnotation("Get Status"),
			mcp.WithDescription(
				"Retrieve detailed information about a specific status in Teamwork Desk by its ID. "+
					"Useful for auditing status usage, troubleshooting ticket workflows, or "+
					"integrating Desk status data into automation workflows."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the status to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			status, err := client.TicketStatuses.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Status retrieved successfully: %s", status.TicketStatus.Name)), nil
		},
	}
}

// StatusList returns a list of statuses that apply to the filters in Teamwork Desk
func StatusList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithTitleAnnotation("List Statuses"),
		mcp.WithOutputSchema[deskmodels.TicketStatusesResponse](),
		mcp.WithDescription(
			"List all statuses in Teamwork Desk, with optional filters for name, color, and code. " +
				"Enables users to audit, analyze, or synchronize status configurations for ticket management, " +
				"reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("name", mcp.Description("The name of the status to filter by.")),
		mcp.WithArray("color", mcp.Description("The color of the status to filter by.")),
		mcp.WithArray("code", mcp.Description("The code of the status to filter by.")),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodStatusList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the status list
			name := request.GetStringSlice("name", []string{})
			color := request.GetStringSlice("color", []string{})
			code := request.GetStringSlice("code", []string{})

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
			setPagination(&params, request)

			statuses, err := client.TicketStatuses.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list statuses: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Statuses retrieved successfully: %v", statuses)), nil
		},
	}
}

// StatusCreate creates a status in Teamwork Desk
func StatusCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodStatusCreate),
			mcp.WithTitleAnnotation("Create Status"),
			mcp.WithDescription(
				"Create a new status in Teamwork Desk by specifying its name, color, and display order. "+
					"Useful for customizing ticket workflows, introducing new resolution states, or "+
					"adapting Desk to evolving support processes."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the status."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the status."),
			),
			mcp.WithNumber("displayOrder", mcp.Description("The display order of the status.")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			status, err := client.TicketStatuses.Create(ctx, &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         request.GetString("name", ""),
					Color:        request.GetString("color", ""),
					DisplayOrder: request.GetInt("displayOrder", 0),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create status: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Status created successfully with ID %d", status.TicketStatus.ID)), nil
		},
	}
}

// StatusUpdate updates a status in Teamwork Desk
func StatusUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodStatusUpdate),
			mcp.WithTitleAnnotation("Update Status"),
			mcp.WithDescription(
				"Update an existing status in Teamwork Desk by ID, allowing changes to its name, color, and display order. "+
					"Supports evolving support policies, rebranding, or correcting status attributes for improved "+
					"ticket handling."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the status to update."),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the status."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the status."),
			),
			mcp.WithNumber("displayOrder",
				mcp.Description("The display order of the status."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, err := client.TicketStatuses.Update(ctx, request.GetInt("id", 0), &deskmodels.TicketStatusResponse{
				TicketStatus: deskmodels.TicketStatus{
					Name:         request.GetString("name", ""),
					Color:        request.GetString("color", ""),
					DisplayOrder: request.GetInt("displayOrder", 0),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create status: %w", err)
			}

			return mcp.NewToolResultText("Status updated successfully"), nil
		},
	}
}
