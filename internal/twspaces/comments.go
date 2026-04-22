package twspaces

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
)

// List of comment methods available in the Teamwork Spaces MCP service.
const (
	MethodCommentCreate toolsets.Method = "twspaces-create_comment"
	MethodCommentUpdate toolsets.Method = "twspaces-update_comment"
	MethodCommentDelete toolsets.Method = "twspaces-delete_comment"
	MethodCommentGet    toolsets.Method = "twspaces-get_comment"
	MethodCommentList   toolsets.Method = "twspaces-list_comments"
)

// CommentGet retrieves a single comment by space, page, and comment ID.
func CommentGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Comment",
				ReadOnlyHint: true,
			},
			Description: "Retrieve a specific comment on a page in Teamwork Spaces by its ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page containing the comment.",
					},
					"commentId": {
						Type:        "integer",
						Description: "The ID of the comment to retrieve.",
					},
				},
				Required: []string{"spaceId", "pageId", "commentId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			comment, err := client.Comments.Get(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				int64(arguments.GetInt("commentId", 0)),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get comment: %w", err)
			}
			return helpers.NewToolResultJSON(comment)
		},
	}
}

// CommentList lists all comments on a page.
func CommentList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Comments",
				ReadOnlyHint: true,
			},
			Description: "List all comments on a page in Teamwork Spaces. Returns top-level comments along " +
				"with their replies, enabling review of discussions and feedback on documentation.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to list comments for.",
					},
				},
				Required: []string{"spaceId", "pageId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			comments, err := client.Comments.List(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				url.Values{},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to list comments: %w", err)
			}
			return helpers.NewToolResultJSON(comments)
		},
	}
}

// CommentCreate creates a new comment on a page.
func CommentCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Comment",
			},
			Description: "Create a new comment on a page in Teamwork Spaces. Supports top-level comments and " +
				"replies to existing comments.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to comment on.",
					},
					"content": {
						Type:        "string",
						Description: "The content of the comment.",
					},
					"parentId": {
						Type:        "integer",
						Description: "The ID of the parent comment (for creating a reply).",
					},
					"isPrivate": {
						Type:        "boolean",
						Description: "Set to true to create a private comment visible only to space members.",
					},
				},
				Required: []string{"spaceId", "pageId", "content"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.CommentCreate{
				Content:   arguments.GetString("content", ""),
				IsPrivate: arguments.GetBool("isPrivate", false),
			}
			if parentID := int64(arguments.GetInt("parentId", 0)); parentID > 0 {
				req.ParentID = &parentID
			}

			comment, err := client.Comments.Create(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				req,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create comment: %w", err)
			}
			return helpers.NewToolResultJSON(comment)
		},
	}
}

// CommentUpdate updates an existing comment.
func CommentUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Comment",
			},
			Description: "Update an existing comment on a page in Teamwork Spaces. Supports modifying content, " +
				"state, and privacy settings.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page containing the comment.",
					},
					"commentId": {
						Type:        "integer",
						Description: "The ID of the comment to update.",
					},
					"content": {
						Type:        "string",
						Description: "The new content of the comment.",
					},
					"state": {
						Type:        "string",
						Description: "The new state of the comment (e.g. \"active\", \"resolved\").",
					},
					"isPrivate": {
						Type:        "boolean",
						Description: "Change the privacy setting of the comment.",
					},
				},
				Required: []string{"spaceId", "pageId", "commentId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.CommentUpdate{}

			if content := arguments.GetString("content", ""); content != "" {
				req.Content = &content
			}
			if state := arguments.GetString("state", ""); state != "" {
				req.State = &state
			}
			if _, hasPrivate := arguments["isPrivate"]; hasPrivate {
				v := arguments.GetBool("isPrivate", false)
				req.IsPrivate = &v
			}

			comment, err := client.Comments.Update(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				int64(arguments.GetInt("commentId", 0)),
				req,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update comment: %w", err)
			}
			return helpers.NewToolResultJSON(comment)
		},
	}
}

// CommentDelete deletes a comment.
func CommentDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentDelete),
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Comment",
			},
			Description: "Delete a comment from a page in Teamwork Spaces. This action is irreversible.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page containing the comment.",
					},
					"commentId": {
						Type:        "integer",
						Description: "The ID of the comment to delete.",
					},
				},
				Required: []string{"spaceId", "pageId", "commentId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			if err := client.Comments.Delete(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				int64(arguments.GetInt("commentId", 0)),
			); err != nil {
				return nil, fmt.Errorf("failed to delete comment: %w", err)
			}
			return helpers.NewToolResultText("Comment deleted successfully"), nil
		},
	}
}
