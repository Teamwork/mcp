package twdesk

import (
	"context"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
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
func MessageCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodMessageCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Message",
			},
			Description: "Send a reply message to a ticket in Teamwork Desk by specifying the ticket ID and message body. " +
				"Useful for automating ticket responses, integrating external communication systems, or " +
				"customizing support workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"ticketID": {
						Type:        "integer",
						Description: "The ID of the ticket that the message will be sent to.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the message.",
					},
				},
				Required: []string{"ticketID", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			_ = client // TODO: use the client to create the message
			_ = ctx
			_ = arguments
			return helpers.NewToolResultTextError("not implemented"), nil
		},
	}
}
