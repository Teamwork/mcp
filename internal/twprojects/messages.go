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
	MethodMessageCreate toolsets.Method = "twprojects-create_message"
	MethodMessageUpdate toolsets.Method = "twprojects-update_message"
	MethodMessageDelete toolsets.Method = "twprojects-delete_message"
	MethodMessageGet    toolsets.Method = "twprojects-get_message"
	MethodMessageList   toolsets.Method = "twprojects-list_messages"
)

const messageDescription = "In the context of Teamwork.com, a message is a structured communication post within a " +
	"project that allows team members to share updates, discuss topics, and document decisions in a centralized, " +
	"threaded format. It includes a title, a detailed message body, and replies from collaborators, all tied to the " +
	"project for clear context and visibility, making it ideal for important discussions that need to be organized " +
	"and easily referenced over time."

var (
	messageGetOutputSchema  *jsonschema.Schema
	messageListOutputSchema *jsonschema.Schema
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
			Description: "Create a new message in Teamwork.com. " + messageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Message",
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
						Type:        "boolean",
						Description: "Whether the current user should be notified about the new message.",
					},
					"notify": {
						Description: "Who should be notified about the new message. Accepts either 'all' or an " +
							"object specifying user, team, or company IDs.",
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

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						messageCreateRequest.Notify = projects.NewMessageNotifyAll()
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
						messageCreateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
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
			Description: "Update an existing message in Teamwork.com. " + messageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Message",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message to update.",
					},
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
						Type:        "boolean",
						Description: "Whether the current user should be notified about the new message.",
					},
					"notify": {
						Description: "Who should be notified about the new message. Accepts either 'all' or an " +
							"object specifying user, team, or company IDs.",
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

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						messageUpdateRequest.Notify = projects.NewMessageNotifyAll()
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
						messageUpdateRequest.Notify = projects.NewMessageNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
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
			Description: "Delete an existing message in Teamwork.com. " + messageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Message",
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
			Description: "Get an existing message in Teamwork.com. " + messageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Message",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the message to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: messageGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageGetRequest projects.MessageGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&messageGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "List messages in Teamwork.com. " + messageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Messages",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter messages by body or title. " +
							"Each word from the search term is used to match against the message body or title. " +
							"The message will be selected if each word of the term matches the message body or title, not " +
							"requiring that the word matches are in the same field.",
					},
					"project_ids": {
						Type:        "array",
						Description: "A list of project IDs to filter messages by projects",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter messages by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match messages that have all the specified tags. " +
							"If false, the search will match messages that have any of the specified tags. " +
							"Defaults to false.",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination of results.",
					},
					"page_size": {
						Type:        "integer",
						Description: "Number of results per page for pagination.",
					},
				},
			},
			OutputSchema: messageListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var messageListRequest projects.MessageListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&messageListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&messageListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericListParam(&messageListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&messageListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&messageListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&messageListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			messageList, err := projects.MessageList(ctx, engine, messageListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list messages")
			}

			encoded, err := json.Marshal(messageList)
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
				StructuredContent: helpers.StructuredWebLinker(ctx, messageList,
					helpers.WebLinkerWithIDPathBuilder("/app/messages"),
				),
			}, nil
		},
	}
}
