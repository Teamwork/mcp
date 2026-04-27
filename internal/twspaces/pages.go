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

// List of page methods available in the Teamwork Spaces MCP service.
const (
	MethodPageCreate    toolsets.Method = "twspaces-create_page"
	MethodPageDuplicate toolsets.Method = "twspaces-duplicate_page"
	MethodPageUpdate    toolsets.Method = "twspaces-update_page"
	MethodPageDelete    toolsets.Method = "twspaces-delete_page"
	MethodPageGet       toolsets.Method = "twspaces-get_page"
	MethodPageList      toolsets.Method = "twspaces-list_pages"
	MethodPageHome      toolsets.Method = "twspaces-get_homepage"
)

// PageGet retrieves a single page by space ID and page ID.
func PageGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Page",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific page within a space in Teamwork Spaces. " +
				"Returns the page content, metadata, tags, and revision information.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to retrieve.",
					},
				},
				Required: []string{"spaceId", "pageId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			page, err := client.Pages.Get(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get page: %w", err)
			}
			return helpers.NewToolResultJSON(page)
		},
	}
}

// PageList returns the page tree for a space.
func PageList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Pages",
				ReadOnlyHint: true,
			},
			Description: "List all pages in a space as a hierarchical tree in Teamwork Spaces. " +
				"Returns the page tree including child pages, useful for understanding content structure " +
				"and navigation.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: paginationOptions(map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space to list pages for.",
					},
				}),
				Required: []string{"spaceId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			params := url.Values{}
			setPagination(&params, arguments)
			pages, err := client.Pages.List(ctx, int64(arguments.GetInt("spaceId", 0)), params)
			if err != nil {
				return nil, fmt.Errorf("failed to list pages: %w", err)
			}
			return helpers.NewToolResultJSON(pages)
		},
	}
}

// PageHome retrieves the homepage of a space.
func PageHome(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageHome),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Space Homepage",
				ReadOnlyHint: true,
			},
			Description: "Retrieve the homepage of a space in Teamwork Spaces. The homepage is the entry point " +
				"and starting page for a space's content.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space to retrieve the homepage for.",
					},
				},
				Required: []string{"spaceId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			page, err := client.Pages.Home(ctx, int64(arguments.GetInt("spaceId", 0)))
			if err != nil {
				return nil, fmt.Errorf("failed to get homepage: %w", err)
			}
			return helpers.NewToolResultJSON(page)
		},
	}
}

// PageCreate creates a new page within a space.
func PageCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Page",
			},
			Description: "Create a new page within a space in Teamwork Spaces. Supports setting title, content, " +
				"parent page, slug, tags, and publishing options.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space to create the page in.",
					},
					"title": {
						Type:        "string",
						Description: "The title of the page.",
					},
					"content": {
						Description: "The HTML content of the page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"parentId": {
						Description: "The ID of the parent page (for creating a sub-page).",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"slug": {
						Description: "A URL-friendly slug for the page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"isPublish": {
						Description: "Set to true to publish the page immediately (default: draft).",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"isRequiredReading": {
						Description: "Mark this page as required reading for space members.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"isFullWidth": {
						Description: "Display the page in full-width layout.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"changeMessage": {
						Description: "A message describing the changes made in this version.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"readerInlineCommentsEnabled": {
						Description: "Allow readers to add inline comments on this page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"spaceId", "title"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.PageCreate{
				Title:   arguments.GetString("title", ""),
				Content: arguments.GetString("content", ""),
				Slug:    arguments.GetString("slug", ""),
			}

			if parentID := int64(arguments.GetInt("parentId", 0)); parentID > 0 {
				req.ParentID = &parentID
			}
			req.IsPublish = arguments.GetBool("isPublish", false)
			req.IsRequiredReading = arguments.GetBool("isRequiredReading", false)
			req.IsFullWidth = arguments.GetBool("isFullWidth", false)
			req.ReaderInlineCommentsEnabled = arguments.GetBool("readerInlineCommentsEnabled", false)
			req.ChangeMessage = arguments.GetString("changeMessage", "")

			page, err := client.Pages.Create(ctx, int64(arguments.GetInt("spaceId", 0)), req)
			if err != nil {
				return nil, fmt.Errorf("failed to create page: %w", err)
			}
			return helpers.NewToolResultJSON(page)
		},
	}
}

// PageDuplicate duplicates an existing page within a space.
func PageDuplicate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageDuplicate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Duplicate Page",
			},
			Description: "Duplicate an existing page within a space in Teamwork Spaces. Creates a copy of the " +
				"page with a new title, optionally under a different parent page.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to duplicate.",
					},
					"title": {
						Type:        "string",
						Description: "The title for the duplicated page.",
					},
					"parentId": {
						Description: "The ID of the parent page for the duplicate (defaults to same parent).",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"slug": {
						Description: "A URL-friendly slug for the duplicated page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"spaceId", "pageId", "title"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.PageDuplicate{
				Title: arguments.GetString("title", ""),
				Slug:  arguments.GetString("slug", ""),
			}
			if parentID := int64(arguments.GetInt("parentId", 0)); parentID > 0 {
				req.ParentID = &parentID
			}

			page, err := client.Pages.Duplicate(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				req,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to duplicate page: %w", err)
			}
			return helpers.NewToolResultJSON(page)
		},
	}
}

// PageUpdate updates an existing page.
func PageUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Page",
			},
			Description: "Update an existing page in Teamwork Spaces. Supports updating title, content, slug, " +
				"parent page, tags, publishing status, and other page attributes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to update.",
					},
					"title": {
						Description: "The new title of the page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"content": {
						Description: "The new HTML content of the page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"parentId": {
						Description: "The ID of the new parent page (to move the page).",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"slug": {
						Description: "A new URL-friendly slug for the page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"isPublish": {
						Description: "Set to true to publish the page, false to revert to draft.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"isRequiredReading": {
						Description: "Mark or unmark this page as required reading.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"isFullWidth": {
						Description: "Toggle full-width layout for this page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"isMinorChange": {
						Description: "Mark this update as a minor change (won't notify watchers).",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"changeMessage": {
						Description: "A message describing the changes made in this version.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"readerInlineCommentsEnabled": {
						Description: "Allow or disallow readers from adding inline comments.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"spaceId", "pageId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.PageUpdate{}

			if title := arguments.GetString("title", ""); title != "" {
				req.Title = &title
			}
			if content := arguments.GetString("content", ""); content != "" {
				req.Content = &content
			}
			if slug := arguments.GetString("slug", ""); slug != "" {
				req.Slug = &slug
			}
			if changeMsg := arguments.GetString("changeMessage", ""); changeMsg != "" {
				req.ChangeMessage = &changeMsg
			}
			if parentID := int64(arguments.GetInt("parentId", 0)); parentID > 0 {
				req.ParentID = &parentID
			}

			// Handle boolean fields only when explicitly provided
			if _, hasPublish := arguments["isPublish"]; hasPublish {
				v := arguments.GetBool("isPublish", false)
				req.IsPublish = &v
			}
			if _, hasReq := arguments["isRequiredReading"]; hasReq {
				v := arguments.GetBool("isRequiredReading", false)
				req.IsRequiredReading = &v
			}
			if _, hasFW := arguments["isFullWidth"]; hasFW {
				v := arguments.GetBool("isFullWidth", false)
				req.IsFullWidth = &v
			}
			if _, hasMinor := arguments["isMinorChange"]; hasMinor {
				v := arguments.GetBool("isMinorChange", false)
				req.IsMinorChange = &v
			}
			if _, hasComments := arguments["readerInlineCommentsEnabled"]; hasComments {
				v := arguments.GetBool("readerInlineCommentsEnabled", false)
				req.ReaderInlineCommentsEnabled = &v
			}

			page, err := client.Pages.Update(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
				req,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update page: %w", err)
			}
			return helpers.NewToolResultJSON(page)
		},
	}
}

// PageDelete deletes a page.
func PageDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageDelete),
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Page",
			},
			Description: "Delete a page from a space in Teamwork Spaces. This action is irreversible.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"spaceId": {
						Type:        "integer",
						Description: "The ID of the space containing the page.",
					},
					"pageId": {
						Type:        "integer",
						Description: "The ID of the page to delete.",
					},
				},
				Required: []string{"spaceId", "pageId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			if err := client.Pages.Delete(ctx,
				int64(arguments.GetInt("spaceId", 0)),
				int64(arguments.GetInt("pageId", 0)),
			); err != nil {
				return nil, fmt.Errorf("failed to delete page: %w", err)
			}
			return helpers.NewToolResultText("Page deleted successfully"), nil
		},
	}
}
