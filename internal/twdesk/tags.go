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
	MethodTagCreate toolsets.Method = "twdesk-create_tag"
	MethodTagUpdate toolsets.Method = "twdesk-update_tag"
	MethodTagGet    toolsets.Method = "twdesk-get_tag"
	MethodTagList   toolsets.Method = "twdesk-list_tags"
)

func init() {
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)
}

// TagGet finds a tag in Teamwork Desk.  This will find it by ID
func TagGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagGet),
			mcp.WithOutputSchema[deskmodels.Tag](),
			mcp.WithDescription(
				"Retrieve detailed information about a specific tag in Teamwork Desk by its ID. "+
					"Useful for auditing tag usage, troubleshooting ticket categorization, or "+
					"integrating Desk tag data into automation workflows."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the tag to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tag, err := client.Tags.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get tag: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tag retrieved successfully: %s", tag.Tag.Name)), nil
		},
	}
}

// TagList returns a list of tags that apply to the filters in Teamwork Desk
func TagList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(
			"List all tags in Teamwork Desk, with optional filters for name, color, and inbox association. " +
				"Enables users to audit, analyze, or synchronize tag configurations for ticket management, " +
				"reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("name", mcp.Description("The name of the tag to filter by.")),
		mcp.WithString("color", mcp.Description("The color of the tag to filter by.")),
		mcp.WithArray("inboxIDs", mcp.Description("The IDs of the inboxes to filter by.")),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the tag list
			name := request.GetString("name", "")
			color := request.GetString("color", "")
			inboxIDs := request.GetIntSlice("inboxIDs", []int{})

			filter := deskclient.NewFilter()
			if name != "" {
				filter = filter.Eq("name", name)
			}
			if color != "" {
				filter = filter.Eq("color", color)
			}
			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", helpers.SliceToAny(inboxIDs))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, request)

			tags, err := client.Tags.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list tags: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tags retrieved successfully: %v", tags)), nil
		},
	}
}

// TagCreate creates a tag in Teamwork Desk
func TagCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagCreate),
			mcp.WithDescription(
				"Create a new tag in Teamwork Desk by specifying its name and color. Useful for customizing "+
					"ticket workflows, introducing new categories, or adapting Desk to evolving support processes."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the tag."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the tag."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tag, err := client.Tags.Create(ctx, &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  request.GetString("name", ""),
					Color: request.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create tag: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tag created successfully with ID %d", tag.Tag.ID)), nil
		},
	}
}

// TagUpdate updates a tag in Teamwork Desk
func TagUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagUpdate),
			mcp.WithDescription(
				"Update an existing tag in Teamwork Desk by ID, allowing changes to its name and color. "+
					"Supports evolving support policies, rebranding, or correcting tag attributes for improved "+
					"ticket handling."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the tag to update."),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the tag."),
			),
			mcp.WithString("color",
				mcp.Description("The color of the tag."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, err := client.Tags.Update(ctx, request.GetInt("id", 0), &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  request.GetString("name", ""),
					Color: request.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create tag: %w", err)
			}

			return mcp.NewToolResultText("Tag updated successfully"), nil
		},
	}
}
