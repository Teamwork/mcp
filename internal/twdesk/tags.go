package twdesk

import (
	"context"
	"fmt"

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
	MethodTagCreate toolsets.Method = "twdesk-create_tag"
	MethodTagUpdate toolsets.Method = "twdesk-update_tag"
	MethodTagDelete toolsets.Method = "twdesk-delete_tag"
	MethodTagGet    toolsets.Method = "twdesk-get_tag"
	MethodTagList   toolsets.Method = "twdesk-list_tags"
)

func init() {
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagDelete)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)
}

// TagCreate creates a tag in Teamwork.com.
func TagCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagCreate),
			mcp.WithDescription("Create a new tag in Teamwork Desk"),
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
