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
	MethodMessageCreate toolsets.Method = "twprojects-create_message"
	MethodMessageUpdate toolsets.Method = "twprojects-update_message"
	MethodMessageDelete toolsets.Method = "twprojects-delete_message"
	MethodMessageGet    toolsets.Method = "twprojects-get_message"
	MethodMessageList   toolsets.Method = "twprojects-list_messages"
)

var (
	messageGetOutputSchema  *jsonschema.Schema
	messageListOutputSchema *jsonschema.Schema
)

// messageOrdering is the order-by vocabulary of the messages list endpoint.
var messageOrdering = newOrdering("messages",
	projects.MessageOrderByCreatedAt,
	projects.MessageOrderByUpdatedAt,
	projects.MessageOrderByCategory,
	projects.MessageOrderByProject,
	projects.MessageOrderByCreatedBy,
	projects.MessageOrderByUnread,
	projects.MessageOrderByID,
)

func init() {
	var err error

	// generate the output schemas only once
	messageGetOutputSchema, err = jsonschema.For[projects.MessageGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MessageGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(messageGetOutputSchema)
	messageListOutputSchema, err = jsonschema.For[projects.MessageListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MessageListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(messageListOutputSchema)
}

// MessageCreate creates a message in Teamwork.com.
func MessageCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageCreate),
			Description: "Create message in a project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Message",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"title": {
						Type:        "string",
						Description: "The title of the message.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to create the message in.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the message.",
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new message.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify":          helpers.NotifySchema("Who to notify of the new message.", false),
					"attachment_refs": attachmentRefsSchema("message"),
				},
				Required: []string{"title", "project_id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageCreateRequest projects.MessageCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&messageCreateRequest.Title, "title"),
				helpers.RequiredParam(&messageCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&messageCreateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			refs, toolResult := parseAttachmentRefs(arguments)
			if toolResult != nil {
				return toolResult, nil
			}
			messageCreateRequest.PendingFileAttachments = refs

			notifyChosen, notifiers, toolResult := parseNotify(arguments, false)
			if toolResult != nil {
				return toolResult, nil
			}
			switch notifyChosen {
			case notifyChoiceGroup:
				messageCreateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				messageCreateRequest.Notify = projects.NewMessageNotifyAll()
			}

			message, err := projects.MessageCreate(ctx, engine, messageCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create message")
			}
			return helpers.NewToolResultText("Message created successfully with ID %d", message.ID), nil
		},
	}
}

// MessageUpdate updates a message in Teamwork.com.
func MessageUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageUpdate),
			Description: "Update message.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Message",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message to update.",
					},
					"title": {
						Description: "The title of the message.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to create the message in.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"body": {
						Description: "The body of the message.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new message.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": helpers.NotifySchema("Who to notify of the message update.", false),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageUpdateRequest projects.MessageUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&messageUpdateRequest.Title, "title"),
				helpers.OptionalPointerParam(&messageUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&messageUpdateRequest.NotifyCurrentUser, "notify_current_user"),
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
				messageUpdateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				messageUpdateRequest.Notify = projects.NewMessageNotifyAll()
			}

			_, err = projects.MessageUpdate(ctx, engine, messageUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update message")
			}
			return helpers.NewToolResultText("Message updated successfully"), nil
		},
	}
}

// MessageDelete deletes a message in Teamwork.com.
func MessageDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageDelete),
			Description: "Delete message.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Message",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageDeleteRequest projects.MessageDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.MessageDelete(ctx, engine, messageDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete message")
			}
			return helpers.NewToolResultText("Message deleted successfully"), nil
		},
	}
}

// MessageGet retrieves a message in Teamwork.com.
func MessageGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageGet),
			Description: "Get message.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Message",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message to get.",
					},
					"fields": helpers.FieldsSchema[projects.Message]("message"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(messageGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageGetRequest projects.MessageGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.Message](&messageGetRequest.Fields.Message, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(messageGetRequest.Fields.Message) > 0 {
				return helpers.NewRawToolResult(ctx, engine, messageGetRequest, "failed to get message",
					helpers.WebLinkerWithIDPathBuilder("/app/messages"),
				)
			}

			message, err := projects.MessageGet(ctx, engine, messageGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get message")
			}

			encoded, err := json.Marshal(message)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/messages"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, message,
					helpers.WebLinkerWithIDPathBuilder("/app/messages"),
				),
			}, nil
		},
	}
}

// MessageList lists messages in Teamwork.com.
func MessageList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMessageList),
			Description: "List project messages (top-level posts). Use twprojects-list_message_replies for thread replies.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Messages",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter messages by body or title. " +
							"Each word from the search term is used to match against the message body or title. " +
							"The message will be selected if each word of the term matches the message body or title, not " +
							"requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "Filter messages by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"tag_ids":        helpers.TagIDsFilterSchema("messages"),
					"match_all_tags": helpers.MatchAllTagsSchema(),
					"order_by":       messageOrdering.orderBySchema(),
					"order_mode":     orderModeSchema(),
					"page":           helpers.PageSchema(),
					"page_size":      helpers.PageSizeSchema(),
					"verbose":        helpers.VerboseSchema(),
					"count_only":     helpers.CountOnlySchema("messages"),
					"fields":         helpers.FieldsSchema[projects.Message]("message"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(messageListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageListRequest projects.MessageListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&messageListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&messageListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericListParam(&messageListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&messageListRequest.Filters.MatchAllTags, "match_all_tags"),
				messageOrdering.param(&messageListRequest.Filters.OrderBy, &messageListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&messageListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&messageListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.Message](&messageListRequest.Filters.Fields.Messages, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose && len(messageListRequest.Filters.Fields.Messages) == 0 {
				messageListRequest.Filters.Fields.Messages = []projects.MessageField{
					projects.MessageFieldID,
					projects.MessageFieldTitle,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, messageListRequest, "failed to count messages")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, messageListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list messages")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list messages"), "failed to list messages")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/messages"))
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
