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
	MethodMessageCreate toolsets.Method = "twdesk-reply_ticket"
)

// MessageCreate replies to a ticket in Teamwork Desk.
func MessageCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodMessageCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Reply to Ticket",
			},
			Description: "Reply to a ticket. Use threadType=note for internal agent notes.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"ticketID": {
						Type:        "integer",
						Description: "The ID of the ticket that the message will be sent to.",
					},
					"threadType": {
						Description: "'message' is a customer-facing reply; 'note' is an internal agent note.",
						Default:     []byte(`"message"`),
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"message", "note"}},
							{Type: "null"},
						},
					},
					"body": {
						Type:        "string",
						Description: "The body of the message.",
					},
					"bcc": {
						Description: "Email addresses to BCC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
					"cc": {
						Description: "Email addresses to CC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"ticketID", "body", "threadType", "bcc", "cc"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			body := arguments.GetString("body", "")
			threadType := arguments.GetString("threadType", "message")
			data := deskmodels.MessageResponse{
				Message: deskmodels.Message{
					Message:    &body,
					ThreadType: &threadType,
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
