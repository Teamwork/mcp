package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	MethodLinkCreate toolsets.Method = "twprojects-create_link"
	MethodLinkUpdate toolsets.Method = "twprojects-update_link"
	MethodLinkDelete toolsets.Method = "twprojects-delete_link"
	MethodLinkGet    toolsets.Method = "twprojects-get_link"
	MethodLinkList   toolsets.Method = "twprojects-list_links"
)

const linkDescription = "In the context of Teamwork.com, a link Link is a saved URL attached to a project, task, or " +
	"other item, allowing users to quickly reference and access external resources (such as documents, tools, or " +
	"websites) directly within their workflow."

var (
	linkGetOutputSchema  *jsonschema.Schema
	linkListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	linkGetOutputSchema, err = jsonschema.For[projects.LinkGetResponse](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[projects.LegacyNumber](): {
				Type:        "string",
				Description: "A numeric value that is returned as a string.",
			},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for LinkGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(linkGetOutputSchema)
	linkListOutputSchema, err = jsonschema.For[projects.LinkListResponse](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[projects.LegacyNumber](): {
				Type:        "string",
				Description: "A numeric value that is returned as a string.",
			},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for LinkListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(linkListOutputSchema)
}

// LinkCreate creates a link in Teamwork.com.
func LinkCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodLinkCreate),
			Description: "Create a new link in Teamwork.com. " + linkDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Link",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"code": {
						Type:        "string",
						Description: "The URL of the link. This is the actual web address that the link points to.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to create the link in.",
					},
					"title": {
						Description: "The title of the link, which provides a brief summary of the purpose of the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the link. Longer text that provides detailed information about the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to associate with the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": {
						Description: "Who should be notified about the new link. Accepts either 'all' or an " +
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
				Required: []string{"project_id", "code"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var linkCreateRequest projects.LinkCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&linkCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&linkCreateRequest.Code, "code"),
				helpers.OptionalPointerParam(&linkCreateRequest.Title, "title"),
				helpers.OptionalPointerParam(&linkCreateRequest.Description, "description"),
				helpers.OptionalCustomNumericListParam(&linkCreateRequest.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&linkCreateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						linkCreateRequest.Notify = projects.NewLinkNotifyAll()
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
						linkCreateRequest.Notify = projects.NewLinkNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
			} else {
				linkCreateRequest.Notify = projects.NewLinkNotifyAll()
			}

			link, err := projects.LinkCreate(ctx, engine, linkCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create link")
			}
			return helpers.NewToolResultText("Link created successfully with ID %d", link.ID), nil
		},
	}
}

// LinkUpdate updates a link in Teamwork.com.
func LinkUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodLinkUpdate),
			Description: "Update an existing link in Teamwork.com. " + linkDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Link",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the link to update.",
					},
					"code": {
						Description: "The URL of the link. This is the actual web address that the link points to.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"title": {
						Description: "The title of the link, which provides a brief summary of the purpose of the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the link. Longer text that provides detailed information about the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to associate with the link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": {
						Description: "Who should be notified about the new link. Accepts either 'all' or an " +
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
			var linkUpdateRequest projects.LinkUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&linkUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&linkUpdateRequest.Code, "code"),
				helpers.OptionalPointerParam(&linkUpdateRequest.Title, "title"),
				helpers.OptionalPointerParam(&linkUpdateRequest.Description, "description"),
				helpers.OptionalCustomNumericListParam(&linkUpdateRequest.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&linkUpdateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case string:
					switch strings.ToLower(value) {
					case "all":
						linkUpdateRequest.Notify = projects.NewLinkNotifyAll()
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
						linkUpdateRequest.Notify = projects.NewLinkNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either string ('all'), " +
						"or an object"), nil
				}
			} else {
				linkUpdateRequest.Notify = projects.NewLinkNotifyAll()
			}

			_, err = projects.LinkUpdate(ctx, engine, linkUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update link")
			}
			return helpers.NewToolResultText("Link updated successfully"), nil
		},
	}
}

// LinkDelete deletes a link in Teamwork.com.
func LinkDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodLinkDelete),
			Description: "Delete an existing link in Teamwork.com. " + linkDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Link",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the link to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var linkDeleteRequest projects.LinkDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&linkDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.LinkDelete(ctx, engine, linkDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete link")
			}
			return helpers.NewToolResultText("Link deleted successfully"), nil
		},
	}
}

// LinkGet retrieves a link in Teamwork.com.
func LinkGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodLinkGet),
			Description: "Get an existing link in Teamwork.com. " + linkDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Link",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the link to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: linkGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var linkGetRequest projects.LinkGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&linkGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			link, err := projects.LinkGet(ctx, engine, linkGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get link")
			}

			encoded, err := json.Marshal(link)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/links"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, link,
					helpers.WebLinkerWithIDPathBuilder("/app/links"),
				),
			}, nil
		},
	}
}

// LinkList lists links in Teamwork.com.
func LinkList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodLinkList),
			Description: "List links in Teamwork.com. " + linkDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Links",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter links by title or description. " +
							"Each word from the search term is used to match against the link title or description. " +
							"The link will be selected if each word of the term matches the link title or description, not " +
							"requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to filter links by.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to filter links by tags",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"match_all_tags": {
						Description: "If true, the search will match links that have all the specified tags. " +
							"If false, the search will match links that have any of the specified tags. " +
							"Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
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
			OutputSchema: linkListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var linkListRequest projects.LinkListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&linkListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&linkListRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalNumericListParam(&linkListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&linkListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&linkListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&linkListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			linkList, err := projects.LinkList(ctx, engine, linkListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list links")
			}

			encoded, err := json.Marshal(linkList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/links"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, linkList,
					helpers.WebLinkerWithIDPathBuilder("/app/links"),
				),
			}, nil
		},
	}
}
