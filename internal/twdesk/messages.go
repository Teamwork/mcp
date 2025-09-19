package twdesk

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodMessageCreate toolsets.Method = "twdesk-create_message"
)

func init() {
	toolsets.RegisterMethod(MethodMessageCreate)
}

// MessageCreate replies to a ticket in Teamwork Desk.  TODO: Still need to
// define the client for this.
func MessageCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMessageCreate),
			mcp.WithDescription(
				"Send a reply message to a ticket in Teamwork Desk by specifying the ticket ID and message body. "+
					"Useful for automating ticket responses, integrating external communication systems, or "+
					"customizing support workflows."),
			mcp.WithNumber("ticketID",
				mcp.Required(),
				mcp.Description("The ID of the ticket that the message will be sent to."),
			),
			mcp.WithString("body",
				mcp.Required(),
				mcp.Description("The body of the message."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_ = client // TODO: use the client to create the message
			_ = ctx
			_ = request
			return nil, errors.New("not implemented")
		},
	}
}
