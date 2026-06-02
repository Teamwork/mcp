package twchat

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
					"type": {
						Description: "Filter by conversation type: \"rooms\" for group/channel conversations, " +
							"\"pair\" for 1:1 direct messages.",
						AnyOf: []*jsonschema.Schema{{Type: "string", Enum: []any{"rooms", "pair"}}, {Type: "null"}},
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
				Type:               arguments.GetString("type", ""),
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

// DMGetOrCreate resolves the 1:1 conversation with a person, creating it if it
// does not exist yet, and returns the conversation.
func DMGetOrCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodDMGetOrCreate),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get or Create Direct Message",
				ReadOnlyHint: true,
			},
			Description: "Get the 1:1 direct-message conversation with a person, creating it if it does not " +
				"exist yet. Returns the conversation (use its id with send_message). Use list_people to find user_id.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"user_id": {
						Type:        "integer",
						Description: "The ID of the person to get the direct-message conversation with.",
					},
				},
				Required: []string{"user_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			req := pairConversationGetRequest{UserID: int64(arguments.GetInt("user_id", 0))}
			return execute(ctx, engine, req, "failed to resolve direct message conversation")
		},
	}
}

// SendDM sends a message directly to a person, resolving (or creating) the 1:1
// conversation first. It is a convenience alias over DMGetOrCreate + MessageSend.
func SendDM(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSendDM),
			Annotations: &mcp.ToolAnnotations{
				Title: "Send Direct Message",
			},
			Description: "Send a direct message to a person, resolving (or creating) the 1:1 conversation " +
				"automatically. Requires user_id and body. Use list_people to find user_id.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"user_id": {
						Type:        "integer",
						Description: "The ID of the person to send the direct message to.",
					},
					"body": {
						Type:        "string",
						Description: "The message text. Supports Markdown.",
					},
				},
				Required: []string{"user_id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}
			userID := int64(arguments.GetInt("user_id", 0))
			body := arguments.GetString("body", "")
			if body == "" {
				return helpers.NewToolResultTextError("body is required"), nil
			}

			// Resolve (or create) the 1:1 conversation, then post the message to it.
			conversationID, errResult, err := pairConversationID(ctx, engine, userID)
			if err != nil {
				return nil, err
			}
			if errResult != nil {
				return errResult, nil
			}

			req := messageSendRequest{ConversationID: conversationID, Body: body}
			return execute(ctx, engine, req, "failed to send direct message")
		},
	}
}

// pairConversationID resolves the 1:1 conversation id with a person. On a
// tool-level failure (API error, unresolvable conversation) it returns a
// non-nil *mcp.CallToolResult for the caller to return directly; a non-nil
// error indicates an internal failure.
func pairConversationID(
	ctx context.Context,
	engine *twapi.Engine,
	userID int64,
) (int64, *mcp.CallToolResult, error) {
	const label = "failed to resolve direct message conversation"

	resp, err := twapi.ExecuteRaw(ctx, engine, pairConversationGetRequest{UserID: userID})
	if err != nil {
		result, handleErr := helpers.HandleAPIError(err, label)
		return 0, result, handleErr
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		result, handleErr := helpers.HandleAPIError(twapi.NewHTTPError(resp, label), label)
		return 0, result, handleErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var parsed struct {
		Conversation struct {
			ID int64 `json:"id"`
		} `json:"conversation"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, nil, fmt.Errorf("failed to decode direct message conversation response: %w", err)
	}
	if parsed.Conversation.ID == 0 {
		return 0, helpers.NewToolResultTextError(
			"could not resolve a direct message conversation for user %d", userID), nil
	}
	return parsed.Conversation.ID, nil, nil
}
