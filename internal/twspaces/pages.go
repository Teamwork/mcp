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
			Description: "Get page. Returns content, metadata, tags, and revision info.",
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
			Description: "List pages in a space as a hierarchical tree.",
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
			Description: "Get a space's homepage.",
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
			Description: "Create page in a space.",
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
			Description: "Duplicate page with a new title.",
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

// insufficientDraftVersion mirrors the Spaces web app's
// INSUFFICIENT_DRAFT_VERSION_NUMBER. A page whose draftVersion is <= 1 has no
// real collaborative editor draft yet: the editor seeds one from the published
// content the first time someone opens it. A draftVersion greater than this
// means an active editor draft exists.
const insufficientDraftVersion = 1

// draftOverwriteWarning is surfaced when an API content/publish write targets a
// page that already has an active editor draft. Content written through the REST
// API updates only the published page, not the collaborative editor draft, so
// the next person who edits the page in the Spaces web app sees the older draft
// and can silently overwrite what was just published. This is a known Spaces
// limitation (the API has no way to update the editor draft), not a problem with
// the arguments supplied to this tool.
const draftOverwriteWarning = "⚠️ Draft-sync warning: this page already has an active editor draft " +
	"(draftVersion=%d). This update changes the PUBLISHED content only — it does NOT update the " +
	"collaborative editor draft (\"Edit version\"). The next time someone opens this page in the Spaces " +
	"web editor they will see the older draft, and saving from there can overwrite the content you just " +
	"published. To resync the draft, open the page in the Spaces web editor and choose \"Revert to last " +
	"published version\" from the ⋯ (more options) menu BEFORE making any edits — this replaces the stale " +
	"draft with the content you just published. Alternatively, make this change in the Spaces web app " +
	"instead. This is a known Spaces limitation and is unrelated to the values you passed."

// PageUpdate updates an existing page.
func PageUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Page",
			},
			Description: "Update page. Note: content and publish changes update the published page only, not the " +
				"live collaborative editor draft; if the page has an active editor draft, re-publishing from the " +
				"Spaces web editor can overwrite these changes (known Spaces limitation).",
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
					"draftVersion": {
						Description: "Optimistic concurrency token for the page's draft content. Required when updating " +
							"`content`; optional otherwise. Obtain the current value from the `draftVersion` field with " +
							"twspaces-get_page or twspaces-list_pages tools.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
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
			if draftVersion := int64(arguments.GetInt("draftVersion", 0)); draftVersion > 0 {
				req.DraftVersion = &draftVersion
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

			spaceID := int64(arguments.GetInt("spaceId", 0))
			pageID := int64(arguments.GetInt("pageId", 0))

			// Detect the known draft-sync limitation before writing: if this write
			// changes the published content and the page already has an active editor
			// draft, the write will not be reflected in that draft. Content updates
			// already require draftVersion, so the risk is usually readable straight
			// from the argument; fall back to a best-effort lookup otherwise.
			_, hasPublish := arguments["isPublish"]
			var draftWarning string
			if req.Content != nil || hasPublish {
				draftVersion := int64(arguments.GetInt("draftVersion", 0))
				if draftVersion <= insufficientDraftVersion {
					if existing, gErr := client.Pages.Get(ctx, spaceID, pageID); gErr == nil &&
						existing != nil && existing.Page.DraftVersion != nil {
						draftVersion = *existing.Page.DraftVersion
					}
				}
				if draftVersion > insufficientDraftVersion {
					draftWarning = fmt.Sprintf(draftOverwriteWarning, draftVersion)
				}
			}

			page, err := client.Pages.Update(ctx, spaceID, pageID, req)
			if err != nil {
				return nil, fmt.Errorf("failed to update page: %w", err)
			}

			result, err := helpers.NewToolResultJSON(page)
			if err != nil {
				return nil, err
			}
			if draftWarning != "" {
				result.Content = append([]mcp.Content{&mcp.TextContent{Text: draftWarning}}, result.Content...)
			}
			return result, nil
		},
	}
}

// PageDelete deletes a page.
func PageDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodPageDelete),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Page",
				DestructiveHint: new(true),
			},
			Description: "Delete page. Irreversible.",
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
