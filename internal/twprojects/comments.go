package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

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
	MethodCommentCreate toolsets.Method = "twprojects-create_comment"
	MethodCommentUpdate toolsets.Method = "twprojects-update_comment"
	MethodCommentDelete toolsets.Method = "twprojects-delete_comment"
	MethodCommentGet    toolsets.Method = "twprojects-get_comment"
	MethodCommentList   toolsets.Method = "twprojects-list_comments"
)

var (
	commentGetOutputSchema  *jsonschema.Schema
	commentListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	commentGetOutputSchema, err = jsonschema.For[projects.CommentGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(commentGetOutputSchema)
	commentListOutputSchema, err = jsonschema.For[projects.CommentListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(commentListOutputSchema)
}

// CommentCreate creates a comment in Teamwork.com.
func CommentCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentCreate),
			Description: "Create comment on a task, milestone, notebook, file, or link.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Comment",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"type": {
								Type:        "string",
								Description: "The type of object to create the comment for.",
								Enum: []any{
									"tasks",
									"milestones",
									"files",
									"notebooks",
									"links",
								},
							},
							"id": {
								Type:        "integer",
								Description: "The ID of the object to create the comment for.",
							},
						},
						Required:    []string{"type", "id"},
						Description: "The object to create the comment for. It can be a tasks, milestones, files or notebooks.",
					},
					"body": {
						Type:        "string",
						Description: "The content of the comment. The content can be added as text or HTML.",
					},
					"content_type": {
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"TEXT", "HTML"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new comment.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": {
						Description: "Who should be notified about the new comment. Accepts either 'all', true (followers) or an " +
							"object specifying user, team, or company IDs. By default, followers are notified.",
						Default: json.RawMessage(`true`),
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
										Type:        "boolean",
										Description: "Notify all followers of the entity this comment is related to.",
										Enum:        []any{true},
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
				Required: []string{"object", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentCreateRequest projects.CommentCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&commentCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentCreateRequest.ContentType, "content_type"),
				helpers.OptionalPointerParam(&commentCreateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case bool:
					if !value {
						return helpers.NewToolResultTextError("invalid parameters: notify must be true when using boolean"), nil
					}
					commentCreateRequest.Notify = projects.NewCommentNotifyFollowers()
				case string:
					switch strings.ToLower(value) {
					case "all":
						commentCreateRequest.Notify = projects.NewCommentNotifyAll()
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
						commentCreateRequest.Notify = projects.NewCommentNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either boolean true, " +
						"string ('all'), or an object"), nil
				}
			} else {
				commentCreateRequest.Notify = projects.NewCommentNotifyFollowers()
			}

			var objectType string
			var objectID int64
			object, ok := arguments["object"]
			if !ok {
				return helpers.NewToolResultTextError("missing required parameter: object"), nil
			}
			objectMap, ok := object.(map[string]any)
			if !ok {
				return helpers.NewToolResultTextError("invalid object: expected an object, got %T", object), nil
			} else if objectMap == nil {
				return helpers.NewToolResultTextError("object cannot be nil"), nil
			}
			err = helpers.ParamGroup(objectMap,
				helpers.RequiredParam(&objectType, "type"),
				helpers.RequiredNumericParam(&objectID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid object: %s", err.Error()), nil
			}

			switch strings.ToLower(objectType) {
			case "tasks":
				commentCreateRequest.Path.TaskID = objectID
			case "milestones":
				commentCreateRequest.Path.MilestoneID = objectID
			case "files":
				commentCreateRequest.Path.FileVersionID = objectID
			case "notebooks":
				commentCreateRequest.Path.NotebookID = objectID
			case "links":
				commentCreateRequest.Path.LinkID = objectID
			default:
				return helpers.NewToolResultTextError("invalid object type: %s", objectType), nil
			}

			comment, err := projects.CommentCreate(ctx, engine, commentCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create comment")
			}
			return helpers.NewToolResultText("Comment created successfully with ID %d", comment.ID), nil
		},
	}
}

// CommentUpdate updates a comment in Teamwork.com.
func CommentUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentUpdate),
			Description: "Update comment.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Comment",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to update.",
					},
					"body": {
						Type:        "string",
						Description: "The content of the comment. The content can be added as text or HTML.",
					},
					"content_type": {
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"TEXT", "HTML"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the comment change.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": {
						Description: "Who should be notified about the comment change. Accepts either 'all', true (followers) or " +
							"an object specifying user, team, or company IDs. By default, followers are notified.",
						Default: json.RawMessage(`true`),
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
										Type:        "boolean",
										Description: "Notify all followers of the entity this comment is related to.",
										Enum:        []any{true},
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
				Required: []string{"id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentUpdateRequest projects.CommentUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentUpdateRequest.Path.ID, "id"),
				helpers.RequiredParam(&commentUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentUpdateRequest.ContentType, "content_type"),
				helpers.OptionalPointerParam(&commentUpdateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if notify, ok := arguments["notify"]; ok {
				switch value := notify.(type) {
				case bool:
					if !value {
						return helpers.NewToolResultTextError("invalid parameters: notify must be true when using boolean"), nil
					}
					commentUpdateRequest.Notify = projects.NewCommentNotifyFollowers()
				case string:
					switch strings.ToLower(value) {
					case "all":
						commentUpdateRequest.Notify = projects.NewCommentNotifyAll()
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
						commentUpdateRequest.Notify = projects.NewCommentNotifyGroup(*notifiers)
					}
				default:
					return helpers.NewToolResultTextError("invalid parameters: notify must be either boolean true, " +
						"string ('all'), or an object"), nil
				}
			} else {
				commentUpdateRequest.Notify = projects.NewCommentNotifyFollowers()
			}

			_, err = projects.CommentUpdate(ctx, engine, commentUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update comment")
			}
			return helpers.NewToolResultText("Comment updated successfully"), nil
		},
	}
}

// CommentDelete deletes a comment in Teamwork.com.
func CommentDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentDelete),
			Description: "Delete comment.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Comment",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentDeleteRequest projects.CommentDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CommentDelete(ctx, engine, commentDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete comment")
			}
			return helpers.NewToolResultText("Comment deleted successfully"), nil
		},
	}
}

// CommentGet retrieves a comment in Teamwork.com.
func CommentGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentGet),
			Description: "Get comment.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Comment",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: commentGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentGetRequest projects.CommentGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			comment, err := projects.CommentGet(ctx, engine, commentGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get comment")
			}

			encoded, err := json.Marshal(comment)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, comment, commentPathBuilder),
			}, nil
		},
	}
}

// CommentList lists comments in Teamwork.com.
func CommentList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentList),
			Description: "List comments. Scope by one of task_id, milestone_id, notebook_id, link_id, or file_version_id; " +
				"omit all for site-wide.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Description: "The ID of the task to retrieve comments for. Provide this to scope comments to a task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"milestone_id": {
						Description: "The ID of the milestone to retrieve comments for. Provide this to scope comments to a milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"notebook_id": {
						Description: "The ID of the notebook to retrieve comments for. Provide this to scope comments to a notebook.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"link_id": {
						Description: "The ID of the link to retrieve comments for. Provide this to scope comments to a link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"file_version_id": {
						Description: "The ID of the file version to retrieve comments for. Each file can have multiple versions, " +
							"and comments can be associated with specific versions.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"search_term": helpers.SearchTermSchema("comments", "name"),
					"updated_after": {
						Description: "Filter comments updated after this date and time. " +
							"The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ. By default it will only return comments " +
							"updated on the last 3 months.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&commentListRequest.Path.TaskID, "task_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.MilestoneID, "milestone_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.NotebookID, "notebook_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.LinkID, "link_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.FileVersionID, "file_version_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalTimeParam(&commentListRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if commentListRequest.Filters.UpdatedAfter.IsZero() {
				// default to last 3 months to improve performance
				commentListRequest.Filters.UpdatedAfter = time.Now().AddDate(0, -3, 0)
			}

			if !verbose {
				commentListRequest.Filters.Fields.Comments = []projects.CommentField{
					projects.CommentFieldID,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list comments"), "failed to list comments")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, commentPathBuilder)
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			if verbose {
				var structured any
				if err := json.Unmarshal(linked, &structured); err != nil {
					return nil, fmt.Errorf("failed to decode response: %w", err)
				}
				result.StructuredContent = structured
			}
			return result, nil
		},
	}
}

func commentPathBuilder(object map[string]any) string {
	id := object["id"]
	var relatedObjectType, relatedObjectID any
	if relatedObject, ok := object["object"]; ok {
		if relatedMap, ok := relatedObject.(map[string]any); ok {
			relatedObjectType = relatedMap["type"]
			relatedObjectID = relatedMap["id"]
		}
	}
	if id == nil || relatedObjectType == nil {
		return ""
	}
	if id == reflect.Zero(reflect.TypeOf(id)).Interface() {
		return ""
	}
	if numeric, ok := id.(float64); ok && math.Trunc(numeric) == numeric {
		id = int64(numeric)
	}
	if relatedObjectType == reflect.Zero(reflect.TypeOf(relatedObjectType)).Interface() {
		return ""
	}
	if relatedObjectID == reflect.Zero(reflect.TypeOf(relatedObjectID)).Interface() {
		return ""
	}
	if numeric, ok := relatedObjectID.(float64); ok && math.Trunc(numeric) == numeric {
		relatedObjectID = int64(numeric)
	}
	return fmt.Sprintf("/#%v/%v?c=%v", relatedObjectType, relatedObjectID, id)
}
