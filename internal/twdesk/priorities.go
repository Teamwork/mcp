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
	MethodPriorityCreate toolsets.Method = "twdesk-create_priority"
	MethodPriorityUpdate toolsets.Method = "twdesk-update_priority"
	MethodPriorityGet    toolsets.Method = "twdesk-get_priority"
	MethodPriorityList   toolsets.Method = "twdesk-list_priorities"
)

func init() {
	toolsets.RegisterMethod(MethodPriorityCreate)
	toolsets.RegisterMethod(MethodPriorityUpdate)
	toolsets.RegisterMethod(MethodPriorityGet)
	toolsets.RegisterMethod(MethodPriorityList)
}

// PriorityGet finds a priority in Teamwork Desk.  This will find it by ID
func PriorityGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodPriorityGet),
			mcp.WithTitleAnnotation("Get Priority"),
			mcp.WithOutputSchema[deskmodels.TicketPriorityResponse](),
			mcp.WithDescription(
				"Retrieve detailed information about a specific priority in Teamwork Desk by its ID. "+
					"Useful for inspecting priority attributes, troubleshooting ticket routing, or "+
					"integrating Desk priority data into automation workflows."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the priority to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			priority, err := client.TicketPriorities.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get priority: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Priority retrieved successfully: %s", priority.TicketPriority.Name)), nil
		},
	}
}

// PriorityList returns a list of priorities that apply to the filters in Teamwork Desk
func PriorityList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithTitleAnnotation("List Priorities"),
		mcp.WithOutputSchema[deskmodels.TicketPrioritiesResponse](),
		mcp.WithDescription(
			"List all available priorities in Teamwork Desk, with optional filters for name and color. " +
				"Enables users to audit, analyze, or synchronize priority configurations for ticket management, " +
				"reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("name",
			mcp.Description("The name of the priority to filter by."),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
		mcp.WithArray("color",
			mcp.Description("The color of the priority to filter by."),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodPriorityList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the priority list
			name := request.GetStringSlice("name", []string{})
			color := request.GetStringSlice("color", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(color) > 0 {
				filter = filter.In("color", helpers.SliceToAny(color))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, request)

			priorities, err := client.TicketPriorities.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list priorities: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Priorities retrieved successfully: %v", priorities)), nil
		},
	}
}

// PriorityCreate creates a priority in Teamwork Desk
func PriorityCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodPriorityCreate),
			mcp.WithTitleAnnotation("Create Priority"),
			mcp.WithDescription(
				"Create a new priority in Teamwork Desk by specifying its name and color. Useful for customizing "+
					"ticket workflows, introducing new escalation levels, or adapting Desk to evolving support processes."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the priority."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the priority."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			priority, err := client.TicketPriorities.Create(ctx, &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  request.GetString("name", ""),
					Color: request.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create priority: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Priority created successfully with ID %d", priority.TicketPriority.ID)), nil
		},
	}
}

// PriorityUpdate updates a priority in Teamwork Desk
func PriorityUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodPriorityUpdate),
			mcp.WithTitleAnnotation("Update Priority"),
			mcp.WithDescription(
				"Update an existing priority in Teamwork Desk by ID, allowing changes to its name and color. "+
					"Supports evolving support policies, rebranding, or correcting priority attributes for improved "+
					"ticket handling."),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the priority to update."),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the priority."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the priority."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, err := client.TicketPriorities.Update(ctx, request.GetInt("id", 0), &deskmodels.TicketPriorityResponse{
				TicketPriority: deskmodels.TicketPriority{
					Name:  request.GetString("name", ""),
					Color: request.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create priority: %w", err)
			}

			return mcp.NewToolResultText("Priority updated successfully"), nil
		},
	}
}
