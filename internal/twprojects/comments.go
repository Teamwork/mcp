package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
}

// CommentCreate creates a comment in Teamwork.com.
func CommentCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentCreate),
			mcp.WithDescription("Create a new comment in Teamwork.com. "+commentDescription),
			mcp.WithTitleAnnotation("Create Comment"),
			mcp.WithObject("object",
				mcp.Required(),
				mcp.Description("The object to create the comment for. It can be a tasks, milestones, files or notebooks."),
				mcp.Properties(map[string]any{
					"type": map[string]any{
						"type":        "string",
						"enum":        []string{"tasks", "milestones", "files", "notebooks"},
						"description": "The type of object to create the comment for.",
					},
					"id": map[string]any{
						"type":        "number",
						"description": "The ID of the object to create the comment for.",
					},
				}),
			),
			mcp.WithString("body",
				mcp.Required(),
				mcp.Description("The content of the comment. The content can be added as text or HTML."),
			),
			mcp.WithString("content_type",
				mcp.Description("The content type of the comment. It can be either 'TEXT' or 'HTML'."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentCreateRequest projects.CommentCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&commentCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentCreateRequest.ContentType, "content_type"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			var objectType string
			var objectID int64
			object, ok := request.GetArguments()["object"]
			if !ok {
				return nil, fmt.Errorf("missing required parameter: object")
			}
			objectMap, ok := object.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid object: expected an object, got %T", object)
			} else if objectMap == nil {
				return nil, fmt.Errorf("object cannot be nil")
			}
			err = helpers.ParamGroup(objectMap,
				helpers.RequiredParam(&objectType, "type"),
				helpers.RequiredNumericParam(&objectID, "id"),
			)
			if err != nil {
				return nil, fmt.Errorf("invalid object: %w", err)
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
				return nil, fmt.Errorf("invalid object type: %s", objectType)
			}

			comment, err := projects.CommentCreate(ctx, engine, commentCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create comment")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Comment created successfully with ID %d", comment.ID)), nil
		},
	}
}

// CommentUpdate updates a comment in Teamwork.com.
func CommentUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentUpdate),
			mcp.WithDescription("Update an existing comment in Teamwork.com. "+commentDescription),
			mcp.WithTitleAnnotation("Update Comment"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the comment to update."),
			),
			mcp.WithString("body",
				mcp.Required(),
				mcp.Description("The content of the comment. The content can be added as text or HTML."),
			),
			mcp.WithString("content_type",
				mcp.Description("The content type of the comment. It can be either 'TEXT' or 'HTML'."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentUpdateRequest projects.CommentUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentUpdateRequest.Path.ID, "id"),
				helpers.RequiredParam(&commentUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentUpdateRequest.ContentType, "content_type"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.CommentUpdate(ctx, engine, commentUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update comment")
			}

			return mcp.NewToolResultText("Comment updated successfully"), nil
		},
	}
}

// CommentDelete deletes a comment in Teamwork.com.
func CommentDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentDelete),
			mcp.WithDescription("Delete an existing comment in Teamwork.com. "+commentDescription),
			mcp.WithTitleAnnotation("Delete Comment"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the comment to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentDeleteRequest projects.CommentDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.CommentDelete(ctx, engine, commentDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete comment")
			}

			return mcp.NewToolResultText("Comment deleted successfully"), nil
		},
	}
}

// CommentGet retrieves a comment in Teamwork.com.
func CommentGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentGet),
			mcp.WithDescription("Get an existing comment in Teamwork.com. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Comment"),
			mcp.WithOutputSchema[projects.CommentGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the comment to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentGetRequest projects.CommentGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			comment, err := projects.CommentGet(ctx, engine, commentGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get comment")
			}

			encoded, err := json.Marshal(comment)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
		},
	}
}

// CommentList lists comments in Teamwork.com.
func CommentList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentList),
			mcp.WithDescription("List comments in Teamwork.com. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Comments"),
			mcp.WithOutputSchema[projects.CommentListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter comments by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
		},
	}
}

// CommentListByFileVersion lists comments by file version in Teamwork.com.
func CommentListByFileVersion(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentListByFileVersion),
			mcp.WithDescription("List comments in Teamwork.com by file version. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Comments by File Version"),
			mcp.WithOutputSchema[projects.CommentListResponse](),
			mcp.WithNumber("file_version_id",
				mcp.Required(),
				mcp.Description("The ID of the file version to retrieve comments for. Each file can have multiple versions, "+
					"and comments can be associated with specific versions."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter comments by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentListRequest.Path.FileVersionID, "file_version_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
		},
	}
}

// CommentListByMilestone lists comments by milestone in Teamwork.com.
func CommentListByMilestone(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentListByMilestone),
			mcp.WithDescription("List comments in Teamwork.com by milestone. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Comments by Milestone"),
			mcp.WithOutputSchema[projects.CommentListResponse](),
			mcp.WithNumber("milestone_id",
				mcp.Required(),
				mcp.Description("The ID of the milestone to retrieve comments for."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter comments by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentListRequest.Path.MilestoneID, "milestone_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
		},
	}
}

// CommentListByNotebook lists comments by notebook in Teamwork.com.
func CommentListByNotebook(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentListByNotebook),
			mcp.WithDescription("List comments in Teamwork.com by notebook. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Comments by Notebook"),
			mcp.WithOutputSchema[projects.CommentListResponse](),
			mcp.WithNumber("notebook_id",
				mcp.Required(),
				mcp.Description("The ID of the notebook to retrieve comments for."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter comments by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentListRequest.Path.NotebookID, "notebook_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
		},
	}
}

// CommentListByTask lists comments by task in Teamwork.com.
func CommentListByTask(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCommentListByTask),
			mcp.WithDescription("List comments in Teamwork.com by task. "+commentDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Comments by Task"),
			mcp.WithOutputSchema[projects.CommentListResponse](),
			mcp.WithNumber("task_id",
				mcp.Required(),
				mcp.Description("The ID of the task to retrieve comments for."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter comments by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&commentListRequest.Path.TaskID, "task_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			commentList, err := projects.CommentList(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}

			encoded, err := json.Marshal(commentList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded, commentPathBuilder))), nil
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
