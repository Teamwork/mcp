package twdesk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskmodels "github.com/teamwork/desksdkgo/models"
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

// MessageCreate replies to a ticket in Teamwork Desk.
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
					"threadType": {
						Type:        "string",
						Description: "The thread type of the message. Valid values are 'message' or 'note'.",
						Enum:        []any{"message", "note"},
						Deprecated:  false,
					},
					"body": {
						Type:        "string",
						Description: "The body of the message.",
					},
					"bcc": {
						Type:        "array",
						Description: "An array of email addresses to BCC on message reply.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
					"cc": {
						Type:        "array",
						Description: "An array of email addresses to CC on message reply.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
				},
				Required: []string{"ticketID", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			data := deskmodels.MessageResponse{
				Message: deskmodels.Message{
					Message:    arguments.GetString("body", ""),
					ThreadType: arguments.GetString("threadType", "message"),
				},
			}

			if len(arguments.GetStringSlice("bcc", []string{})) > 0 {
				data.Message.BCC = arguments.GetStringSlice("bcc", []string{})
			}

			if len(arguments.GetStringSlice("cc", []string{})) > 0 {
				data.Message.CC = arguments.GetStringSlice("cc", []string{})
			}

			message, err := client.Messages.CreateForTicket(ctx, arguments.GetInt("ticketID", 0), &data)
			if err != nil {
				return nil, fmt.Errorf("failed to create message: %w", err)
			}

			return helpers.NewToolResultJSON(message)
		},
	}
}
