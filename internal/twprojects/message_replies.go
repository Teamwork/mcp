package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

const messageReplyDescription = "In the context of Teamwork.com, a message reply is a response within a project " +
	"message thread that allows team members to contribute to the discussion, ask questions, or provide updates while " +
	"keeping all communication organized under the original message. Replies maintain context by staying linked to the " +
	"main topic, include the author and timestamp, and help create a clear, ongoing conversation that is easy for " +
	"everyone involved to follow and reference."

var (
	messageReplyGetOutputSchema  *jsonschema.Schema
	messageReplyListOutputSchema *jsonschema.Schema
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
			Description: "Create a new message reply in Teamwork.com. " + messageReplyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Message Reply",
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
					"notify": {
						Description: "Who should be notified about the new message reply. Accepts either 'all' or an " +
							"object specifying user, team, or company IDs. By default, all project members are notified.",
						Default: json.RawMessage(`"all"`),
						AnyOf: []*jsonschema.Schema{
							{
								AnyOf: []*jsonschema.Schema{
									{
										Type:        "string",
										Description: "Notify all project members.",
										Enum: []any{
											"all",
										},
									},
									{
										Type: "object",
										Description: "An object containing the users, teams or companies to notify. At least one of the " +
											"properties (user_ids, team_ids, company_ids) is required.",
										Properties: map[string]*jsonschema.Schema{
											"user_ids": {
												Type:        "array",
												Description: "List of user IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
											"company_ids": {
												Type:        "array",
												Description: "List of company IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
											"team_ids": {
												Type:        "array",
												Description: "List of team IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
										},
										MinProperties: new(1),
										MaxProperties: new(3),
										AnyOf: []*jsonschema.Schema{
											{Required: []string{"user_ids"}},
											{Required: []string{"company_ids"}},
											{Required: []string{"team_ids"}},
										},
									},
								},
							},
							{Type: "null"},
						},
					},
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

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						messageReplyCreateRequest.Notify = projects.NewMessageNotifyAll()
					default:
						return helpers.NewToolResultTextError("invalid parameters: notify must be 'all'"), nil
					}
				case map[string]any:
					if notifiers, toolResult := parseLegacyUserGroups(
						arguments,
						"notify",
						"notifiers",
					); toolResult != nil {
						return toolResult, nil
					} else if notifiers != nil {
						messageReplyCreateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
			} else {
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
			Description: "Update an existing message reply in Teamwork.com. " + messageReplyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update MessageReply",
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
					"notify": {
						Description: "Who should be notified about the new messageReply. Accepts either 'all' or an " +
							"object specifying user, team, or company IDs. By default, all project members are notified.",
						Default: json.RawMessage(`"all"`),
						AnyOf: []*jsonschema.Schema{
							{
								AnyOf: []*jsonschema.Schema{
									{
										Type:        "string",
										Description: "Notify all project members.",
										Enum: []any{
											"all",
										},
									},
									{
										Type: "object",
										Description: "An object containing the users, teams or companies to notify. At least one of the " +
											"properties (user_ids, team_ids, company_ids) is required.",
										Properties: map[string]*jsonschema.Schema{
											"user_ids": {
												Type:        "array",
												Description: "List of user IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
											"company_ids": {
												Type:        "array",
												Description: "List of company IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
											"team_ids": {
												Type:        "array",
												Description: "List of team IDs to notify.",
												Items:       &jsonschema.Schema{Type: "integer"},
												MinItems:    new(1),
											},
										},
										MinProperties: new(1),
										MaxProperties: new(3),
										AnyOf: []*jsonschema.Schema{
											{Required: []string{"user_ids"}},
											{Required: []string{"company_ids"}},
											{Required: []string{"team_ids"}},
										},
									},
								},
							},
							{Type: "null"},
						},
					},
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

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						messageReplyUpdateRequest.Notify = projects.NewMessageNotifyAll()
					default:
						return helpers.NewToolResultTextError("invalid parameters: notify must be 'all'"), nil
					}
				case map[string]any:
					if notifiers, toolResult := parseLegacyUserGroups(
						arguments,
						"notify",
						"notifiers",
					); toolResult != nil {
						return toolResult, nil
					} else if notifiers != nil {
						messageReplyUpdateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
			} else {
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
			Description: "Delete an existing message reply in Teamwork.com. " + messageReplyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Message Reply",
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
			Description: "Get an existing message reply in Teamwork.com. " + messageReplyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Message Reply",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message reply to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: messageReplyGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyGetRequest projects.MessageReplyGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageReplyGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "List message replies in Teamwork.com. " + messageReplyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Message Replies",
				ReadOnlyHint: true,
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
						Description: "A list of message IDs to filter message replies by messages",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "A list of project IDs to filter message replies by projects",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"page": {
						Description: "Page number for pagination of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page_size": {
						Description: "Number of results per page for pagination.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{},
			},
			OutputSchema: messageReplyListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageReplyListRequest projects.MessageReplyListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&messageReplyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&messageReplyListRequest.Filters.MessageIDs, "message_ids"),
				helpers.OptionalNumericListParam(&messageReplyListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericParam(&messageReplyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&messageReplyListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			messageReplyList, err := projects.MessageReplyList(ctx, engine, messageReplyListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list message replies")
			}

			encoded, err := json.Marshal(messageReplyList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: messageReplyList,
			}, nil
		},
	}
}
