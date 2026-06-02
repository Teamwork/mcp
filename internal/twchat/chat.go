package twchat

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// execute runs the request through the shared engine and streams the raw JSON
// response body back to the caller. label is used in error messages.
func execute(
	ctx context.Context,
	engine *twapi.Engine,
	req twapi.HTTPRequester,
	label string,
) (*mcp.CallToolResult, error) {
	resp, err := twapi.ExecuteRaw(ctx, engine, req)
	if err != nil {
		return helpers.HandleAPIError(err, label)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return helpers.HandleAPIError(twapi.NewHTTPError(resp, label), label)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(body)},
		},
	}, nil
}

// CurrentUserGet returns the current authenticated Teamwork Chat user.
func CurrentUserGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCurrentUserGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Current Chat User",
				ReadOnlyHint: true,
			},
			Description: "Get the current authenticated Teamwork Chat user, including identity, " +
				"counts (unread conversations/messages, mentions), and settings.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
				Required:   []string{},
			},
		},
		Handler: func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return execute(ctx, engine, currentUserGetRequest{}, "failed to get current chat user")
		},
	}
}

// ConversationList lists Teamwork Chat conversations for the current user.
func ConversationList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodConversationList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Chat Conversations",
				ReadOnlyHint: true,
			},
			Description: "List Teamwork Chat conversations the current user is a member of.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "Filter conversations by title (substring match).",
						AnyOf:       []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"status": {
						Description: "Filter by conversation status.",
						AnyOf:       []*jsonschema.Schema{{Type: "string", Enum: []any{"all", "active"}}, {Type: "null"}},
					},
					"sort": {
						Description: "Sort order for the returned conversations.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"name", "lastActivityAt", "createdAt", "updatedAt", "relevance"}},
							{Type: "null"},
						},
					},
					"include_message_data": {
						Description: "Include the latest message in each conversation.",
						AnyOf:       []*jsonschema.Schema{{Type: "boolean"}, {Type: "null"}},
					},
					"page_offset": {
						Description: "Zero-based pagination offset.",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"page_limit": {
						Description: "Number of conversations to return (max 10).",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
				},
				Required: []string{},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			req := conversationListRequest{
				PageOffset:         arguments.GetInt("page_offset", 0),
				PageLimit:          arguments.GetInt("page_limit", 0),
				SearchTerm:         arguments.GetString("search_term", ""),
				Status:             arguments.GetString("status", ""),
				Sort:               arguments.GetString("sort", ""),
				IncludeMessageData: arguments.GetBool("include_message_data", false),
			}
			return execute(ctx, engine, req, "failed to list chat conversations")
		},
	}
}

// ConversationGet retrieves a single Teamwork Chat conversation by ID.
func ConversationGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodConversationGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Chat Conversation",
				ReadOnlyHint: true,
			},
			Description: "Get a single Teamwork Chat conversation by ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"conversation_id": {
						Type:        "integer",
						Description: "The ID of the conversation to retrieve.",
					},
				},
				Required: []string{"conversation_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			req := conversationGetRequest{ID: int64(arguments.GetInt("conversation_id", 0))}
			return execute(ctx, engine, req, "failed to get chat conversation")
		},
	}
}

// MessageList lists messages within a Teamwork Chat conversation.
func MessageList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodMessageList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Chat Messages",
				ReadOnlyHint: true,
			},
			Description: "List messages within a Teamwork Chat conversation. Requires conversation_id.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"conversation_id": {
						Type:        "integer",
						Description: "The ID of the conversation to read messages from.",
					},
					"search_term": {
						Description: "Filter messages by text content.",
						AnyOf:       []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"page": {
						Description: "One-based page number.",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"page_size": {
						Description: "Number of messages per page (1-200).",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"before_message_id": {
						Description: "Return messages older than this message ID (cursor).",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"after_message_id": {
						Description: "Return messages newer than this message ID (cursor).",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"created_before": {
						Description: "Return messages created before this time.",
						Examples:    []any{"2023-12-31T23:59:59Z"},
						AnyOf:       []*jsonschema.Schema{{Type: "string", Format: "date-time"}, {Type: "null"}},
					},
					"created_after": {
						Description: "Return messages created after this time.",
						Examples:    []any{"2023-01-01T00:00:00Z"},
						AnyOf:       []*jsonschema.Schema{{Type: "string", Format: "date-time"}, {Type: "null"}},
					},
				},
				Required: []string{"conversation_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			req := messageListRequest{
				ConversationID:  int64(arguments.GetInt("conversation_id", 0)),
				Page:            arguments.GetInt("page", 0),
				PageSize:        arguments.GetInt("page_size", 0),
				SearchTerm:      arguments.GetString("search_term", ""),
				BeforeMessageID: int64(arguments.GetInt("before_message_id", 0)),
				AfterMessageID:  int64(arguments.GetInt("after_message_id", 0)),
				CreatedBefore:   arguments.GetString("created_before", ""),
				CreatedAfter:    arguments.GetString("created_after", ""),
			}
			return execute(ctx, engine, req, "failed to list chat messages")
		},
	}
}

// PeopleList lists people in the Teamwork Chat installation.
func PeopleList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPeopleList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Chat People",
				ReadOnlyHint: true,
			},
			Description: "List people in the Teamwork Chat installation. Useful for resolving names to user IDs.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "Filter people by name or email.",
						AnyOf:       []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"page_offset": {
						Description: "Zero-based pagination offset.",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"page_limit": {
						Description: "Number of people to return.",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
				},
				Required: []string{},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			req := peopleListRequest{
				PageOffset: arguments.GetInt("page_offset", 0),
				PageLimit:  arguments.GetInt("page_limit", 0),
				SearchTerm: arguments.GetString("search_term", ""),
			}
			return execute(ctx, engine, req, "failed to list chat people")
		},
	}
}

// MessageSend posts a message to a Teamwork Chat conversation.
func MessageSend(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodMessageSend),
			Annotations: &mcp.ToolAnnotations{
				Title: "Send Chat Message",
			},
			Description: "Send a message to a Teamwork Chat conversation. Requires conversation_id and body.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"conversation_id": {
						Type:        "integer",
						Description: "The ID of the conversation to post the message to.",
					},
					"body": {
						Type:        "string",
						Description: "The message text. Supports Markdown.",
					},
				},
				Required: []string{"conversation_id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			body := arguments.GetString("body", "")
			if body == "" {
				return helpers.NewToolResultTextError("body is required"), nil
			}
			req := messageSendRequest{
				ConversationID: int64(arguments.GetInt("conversation_id", 0)),
				Body:           body,
			}
			return execute(ctx, engine, req, "failed to send chat message")
		},
	}
}
