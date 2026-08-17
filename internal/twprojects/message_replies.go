package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodMessageReplyCreate toolsets.Method = "twprojects-create_message_reply"
	MethodMessageReplyUpdate toolsets.Method = "twprojects-update_message_reply"
	MethodMessageReplyDelete toolsets.Method = "twprojects-delete_message_reply"
	MethodMessageReplyGet    toolsets.Method = "twprojects-get_message_reply"
	MethodMessageReplyList   toolsets.Method = "twprojects-list_message_replies"
)

var (
	messageReplyGetOutputSchema  *jsonschema.Schema
	messageReplyListOutputSchema *jsonschema.Schema
)

// messageReplyOrdering is the order-by vocabulary of the message replies list endpoint.
var messageReplyOrdering = newOrdering("message replies",
	projects.MessageReplyOrderByCreatedAt,
	projects.MessageReplyOrderByID,
)

func init() {
	var err error

	// generate the output schemas only once
	messageReplyGetOutputSchema, err = jsonschema.For[projects.MessageReplyGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MessageReplyGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(messageReplyGetOutputSchema)
	messageReplyListOutputSchema, err = jsonschema.For[projects.MessageReplyListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MessageReplyListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(messageReplyListOutputSchema)
}

// MessageReplyCreate creates a message reply in Teamwork.com.
func MessageReplyCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageReplyCreate),
			Description: "Create message reply.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Message Reply",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"message_id": {
						Type:        "integer",
						Description: "The ID of the message to create the reply in.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the message reply.",
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new message reply.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": helpers.NotifySchema("Who to notify of the new reply.", false),
				},
				Required: []string{"message_id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyCreateRequest projects.MessageReplyCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageReplyCreateRequest.Path.MessageID, "message_id"),
				helpers.RequiredParam(&messageReplyCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&messageReplyCreateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			notifyChosen, notifiers, toolResult := parseNotify(arguments, false)
			if toolResult != nil {
				return toolResult, nil
			}
			switch notifyChosen {
			case notifyChoiceGroup:
				messageReplyCreateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				messageReplyCreateRequest.Notify = projects.NewMessageNotifyAll()
			}

			messageReply, err := projects.MessageReplyCreate(ctx, engine, messageReplyCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create message reply")
			}
			return helpers.NewToolResultText("Message reply created successfully with ID %d", messageReply.ID), nil
		},
	}
}

// MessageReplyUpdate updates a message reply in Teamwork.com.
func MessageReplyUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageReplyUpdate),
			Description: "Update message reply.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update MessageReply",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message reply to update.",
					},
					"body": {
						Description: "The body of the message reply.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new messageReply.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": helpers.NotifySchema("Who to notify of the reply update.", false),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyUpdateRequest projects.MessageReplyUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageReplyUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&messageReplyUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&messageReplyUpdateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			notifyChosen, notifiers, toolResult := parseNotify(arguments, false)
			if toolResult != nil {
				return toolResult, nil
			}
			switch notifyChosen {
			case notifyChoiceGroup:
				messageReplyUpdateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				messageReplyUpdateRequest.Notify = projects.NewMessageNotifyAll()
			}

			_, err = projects.MessageReplyUpdate(ctx, engine, messageReplyUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update message reply")
			}
			return helpers.NewToolResultText("Message reply updated successfully"), nil
		},
	}
}

// MessageReplyDelete deletes a message reply in Teamwork.com.
func MessageReplyDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageReplyDelete),
			Description: "Delete message reply.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Message Reply",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message reply to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyDeleteRequest projects.MessageReplyDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageReplyDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.MessageReplyDelete(ctx, engine, messageReplyDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete message reply")
			}
			return helpers.NewToolResultText("Message reply deleted successfully"), nil
		},
	}
}

// MessageReplyGet retrieves a message reply in Teamwork.com.
func MessageReplyGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageReplyGet),
			Description: "Get message reply.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Message Reply",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message reply to get.",
					},
					"fields": helpers.FieldsSchema[projects.MessageReply]("message reply"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(messageReplyGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyGetRequest projects.MessageReplyGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageReplyGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.MessageReply](&messageReplyGetRequest.Fields.MessageReply, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(messageReplyGetRequest.Fields.MessageReply) > 0 {
				return helpers.NewRawToolResult(ctx, engine, messageReplyGetRequest, "failed to get message reply", nil)
			}

			messageReply, err := projects.MessageReplyGet(ctx, engine, messageReplyGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get message reply")
			}

			encoded, err := json.Marshal(messageReply)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: messageReply,
			}, nil
		},
	}
}

// MessageReplyList lists message replies in Teamwork.com.
func MessageReplyList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageReplyList),
			Description: "List replies under a message thread. Filter by message_ids or project_ids.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Message Replies",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter message replies by body or title. " +
							"Each word from the search term is used to match against the message reply body or title. " +
							"The message reply will be selected if each word of the term matches the message reply body or title, " +
							"not requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"message_ids": {
						Description: "Filter by message.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "Filter by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"order_by":   messageReplyOrdering.orderBySchema(),
					"order_mode": orderModeSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("message replies"),
					"fields":     helpers.FieldsSchema[projects.MessageReply]("message reply"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(messageReplyListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyListRequest projects.MessageReplyListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&messageReplyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&messageReplyListRequest.Filters.MessageIDs, "message_ids"),
				helpers.OptionalNumericListParam(&messageReplyListRequest.Filters.ProjectIDs, "project_ids"),
				messageReplyOrdering.param(&messageReplyListRequest.Filters.OrderBy, &messageReplyListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&messageReplyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&messageReplyListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.MessageReply](&messageReplyListRequest.Filters.Fields.MessageReplies, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose && len(messageReplyListRequest.Filters.Fields.MessageReplies) == 0 {
				messageReplyListRequest.Filters.Fields.MessageReplies = []projects.MessageReplyField{
					projects.MessageReplyFieldID,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, messageReplyListRequest, "failed to count message replies")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, messageReplyListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list message replies")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list message replies"),
					"failed to list message replies",
				)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
