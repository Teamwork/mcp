package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	MethodCommentCreate            toolsets.Method = "twprojects-create_comment"
	MethodCommentUpdate            toolsets.Method = "twprojects-update_comment"
	MethodCommentDelete            toolsets.Method = "twprojects-delete_comment"
	MethodCommentGet               toolsets.Method = "twprojects-get_comment"
	MethodCommentList              toolsets.Method = "twprojects-list_comments"
	MethodCommentListByFileVersion toolsets.Method = "twprojects-list_comments_by_file_version"
	MethodCommentListByMilestone   toolsets.Method = "twprojects-list_comments_by_milestone"
	MethodCommentListByNotebook    toolsets.Method = "twprojects-list_comments_by_notebook"
	MethodCommentListByTask        toolsets.Method = "twprojects-list_comments_by_task"
)

const commentDescription = "In the Teamwork.com context, a comment is a way for users to communicate and collaborate " +
	"directly within tasks, milestones, files, or other project items. Comments allow team members to provide updates, " +
	"ask questions, give feedback, or share relevant information in a centralized and contextual manner. They support " +
	"rich text formatting, file attachments, and @mentions to notify specific users or teams, helping keep " +
	"discussions organized and easily accessible within the project. Comments are visible to all users with access to " +
	"the item, promoting transparency and keeping everyone aligned."

var (
	commentGetOutputSchema  *jsonschema.Schema
	commentListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodCommentCreate)
	toolsets.RegisterMethod(MethodCommentUpdate)
	toolsets.RegisterMethod(MethodCommentDelete)
	toolsets.RegisterMethod(MethodCommentGet)
	toolsets.RegisterMethod(MethodCommentList)
	toolsets.RegisterMethod(MethodCommentListByFileVersion)
	toolsets.RegisterMethod(MethodCommentListByMilestone)
	toolsets.RegisterMethod(MethodCommentListByNotebook)
	toolsets.RegisterMethod(MethodCommentListByTask)

	var err error

	// generate the output schemas only once
	commentGetOutputSchema, err = jsonschema.For[projects.CommentGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentGetResponse: %v", err))
	}
	commentListOutputSchema, err = jsonschema.For[projects.CommentListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentListResponse: %v", err))
	}
}

// CommentCreate creates a comment in Teamwork.com.
func CommentCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentCreate),
			Description: "Create a new comment in Teamwork.com. " + commentDescription,
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
						Type:        "string",
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						Enum: []any{
							"TEXT",
							"HTML",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&commentCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentCreateRequest.ContentType, "content_type"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			var objectType string
			var objectID int64
			object, ok := arguments["object"]
			if !ok {
				return helpers.NewToolResultTextError("missing required parameter: object"), nil
			}
			objectMap, ok := object.(map[string]any)
			if !ok {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid object: expected an object, got %T", object)), nil
			} else if objectMap == nil {
				return helpers.NewToolResultTextError("object cannot be nil"), nil
			}
			err = helpers.ParamGroup(objectMap,
				helpers.RequiredParam(&objectType, "type"),
				helpers.RequiredNumericParam(&objectID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid object: %s", err.Error())), nil
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
			default:
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid object type: %s", objectType)), nil
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
			Description: "Update an existing comment in Teamwork.com. " + commentDescription,
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
						Type:        "string",
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						Enum: []any{
							"TEXT",
							"HTML",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentUpdateRequest.Path.ID, "id"),
				helpers.RequiredParam(&commentUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentUpdateRequest.ContentType, "content_type"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Delete an existing comment in Teamwork.com. " + commentDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Get an existing comment in Teamwork.com. " + commentDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
				StructuredContent: comment,
			}, nil
		},
	}
}

// CommentList lists comments in Teamwork.com.
func CommentList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentList),
			Description: "List comments in Teamwork.com. " + commentDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type:        "string",
						Description: "A search term to filter comments by name.",
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
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: commentList,
			}, nil
		},
	}
}

// CommentListByFileVersion lists comments by file version in Teamwork.com.
func CommentListByFileVersion(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentListByFileVersion),
			Description: "List comments in Teamwork.com by file version. " + commentDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments by File Version",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"file_version_id": {
						Type: "integer",
						Description: "The ID of the file version to retrieve comments for. Each file can have multiple versions, " +
							"and comments can be associated with specific versions.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter comments by name.",
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
				Required: []string{"file_version_id"},
			},
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentListRequest.Path.FileVersionID, "file_version_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: commentList,
			}, nil
		},
	}
}

// CommentListByMilestone lists comments by milestone in Teamwork.com.
func CommentListByMilestone(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentListByMilestone),
			Description: "List comments in Teamwork.com by milestone. " + commentDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments by Milestone",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone to retrieve comments for.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter comments by name.",
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
				Required: []string{"milestone_id"},
			},
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentListRequest.Path.MilestoneID, "milestone_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: commentList,
			}, nil
		},
	}
}

// CommentListByNotebook lists comments by notebook in Teamwork.com.
func CommentListByNotebook(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentListByNotebook),
			Description: "List comments in Teamwork.com by notebook. " + commentDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments by Notebook",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"notebook_id": {
						Type:        "integer",
						Description: "The ID of the notebook to retrieve comments for.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter comments by name.",
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
				Required: []string{"notebook_id"},
			},
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentListRequest.Path.NotebookID, "notebook_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: commentList,
			}, nil
		},
	}
}

// CommentListByTask lists comments by task in Teamwork.com.
func CommentListByTask(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentListByTask),
			Description: "List comments in Teamwork.com by task. " + commentDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments by Task",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Type:        "integer",
						Description: "The ID of the task to retrieve comments for.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter comments by name.",
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
				Required: []string{"task_id"},
			},
			OutputSchema: commentListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentListRequest.Path.TaskID, "task_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: commentList,
			}, nil
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
