package twdesk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
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

func init() {
	toolsets.RegisterMethod(MethodTypeCreate)
	toolsets.RegisterMethod(MethodTypeUpdate)
	toolsets.RegisterMethod(MethodTypeGet)
	toolsets.RegisterMethod(MethodTypeList)
}

// TypeGet finds a type in Teamwork Desk.  This will find it by ID
func TypeGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTypeGet),
			mcp.WithDescription("Get a type from Teamwork Desk"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the type to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			t, err := client.TicketTypes.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get type: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Type retrieved successfully: %s", t.TicketType.Name)), nil
		},
	}
}

// TypeList returns a list of types that apply to the filters in Teamwork Desk
func TypeList(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTypeList),
			mcp.WithDescription("List all types in Teamwork Desk"),
			mcp.WithArray("name", mcp.Description("The name of the type to filter by.")),
			mcp.WithArray("inboxIDs", mcp.Description("The inbox IDs of the type to filter by.")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the type list
			name := request.GetStringSlice("name", []string{})
			inboxIDs := request.GetStringSlice("inboxIDs", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", name)
			}
			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", inboxIDs)
			}

			params := url.Values{}
			params.Set("filter", filter.Build())

			types, err := client.TicketTypes.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list types: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Types retrieved successfully: %v", types)), nil
		},
	}
}

// TypeCreate creates a type in Teamwork Desk
func TypeCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTypeCreate),
			mcp.WithDescription("Create a new type in Teamwork Desk"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the type."),
			),
			mcp.WithNumber("displayOrder", mcp.Description("The display order of the type.")),
			mcp.WithBoolean("enabledForFutureInboxes",
				mcp.Description("Whether the type is enabled for future inboxes."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			t, err := client.TicketTypes.Create(ctx, &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    request.GetString("name", ""),
					DisplayOrder:            request.GetInt("displayOrder", 0),
					EnabledForFutureInboxes: request.GetBool("enabledForFutureInboxes", false),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create type: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Type created successfully with ID %d", t.TicketType.ID)), nil
		},
	}
}

// TypeUpdate updates a type in Teamwork Desk
func TypeUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTypeUpdate),
			mcp.WithDescription("Update an existing type in Teamwork Desk"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the type to update."),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the type."),
			),
			mcp.WithNumber("displayOrder",
				mcp.Description("The display order of the type."),
			),
			mcp.WithBoolean("enabledForFutureInboxes",
				mcp.Description("Whether the type is enabled for future inboxes."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, err := client.TicketTypes.Update(ctx, request.GetInt("id", 0), &deskmodels.TicketTypeResponse{
				TicketType: deskmodels.TicketType{
					Name:                    request.GetString("name", ""),
					DisplayOrder:            request.GetInt("displayOrder", 0),
					EnabledForFutureInboxes: request.GetBool("enabledForFutureInboxes", false),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create type: %w", err)
			}

			return mcp.NewToolResultText("Type updated successfully"), nil
		},
	}
}
