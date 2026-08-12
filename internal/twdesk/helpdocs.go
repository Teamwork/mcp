package twdesk

import (
	"context"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskmodels "github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodHelpDocArticleCreate toolsets.Method = "twdesk-create_helpdoc_article"
	MethodHelpDocArticleUpdate toolsets.Method = "twdesk-update_helpdoc_article"
	MethodHelpDocArticleGet    toolsets.Method = "twdesk-get_helpdoc_article"
	MethodHelpDocArticleSearch toolsets.Method = "twdesk-search_helpdoc_articles"
)

// HelpDocArticleGet retrieves a single help doc article by ID.
func HelpDocArticleGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocArticleGet),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Help Doc Article",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Get a help doc article by ID.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the help doc article to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id", "fields"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			article, err := client.HelpDocArticles.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get help doc article")
			}
			return helpers.NewToolResultJSON(article)
		},
	}
}

// HelpDocArticleSearch searches help doc articles using the dedicated search API.
func HelpDocArticleSearch(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocArticleSearch),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Search Help Doc Articles",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Search help doc articles. Filter by search term, status, site, or category.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"search": {
						Description: "Free-text search term matched against article title and content.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"status": {
						Description: "Filter by article status (e.g. \"published\", \"draft\").",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"siteID": {
						Description: "Filter by help doc site ID. Use twdesk-list_helpdoc_sites to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"categoryID": {
						Description: "Filter by help doc category ID.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page": {
						Description: "Page number (1-based).",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"pageSize": {
						Description: "Number of results per page.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"search", "status", "siteID", "categoryID", "page", "pageSize"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			filter := &deskmodels.SearchHelpdocsFilter{
				Search:     arguments.GetString("search", ""),
				Status:     arguments.GetString("status", ""),
				SiteID:     int64(arguments.GetInt("siteID", 0)),
				CategoryID: int64(arguments.GetInt("categoryID", 0)),
				Page:       arguments.GetInt("page", 1),
				PageSize:   arguments.GetInt("pageSize", 10),
			}

			articles, err := client.HelpDocArticles.Search(ctx, filter)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to search help doc articles")
			}
			return helpers.NewToolResultJSON(articles)
		},
	}
}

// HelpDocArticleCreate creates a new help doc article.
func HelpDocArticleCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocArticleCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Help Doc Article",
				// A published, non-private article is visible on the public knowledge
				// base, so this can change publicly-visible internet state.
				OpenWorldHint:   new(true),
				DestructiveHint: new(false),
			},
			Description: "Create a new help doc article.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"siteID": {
						Type: "integer",
						Description: "The ID of the help doc site to create the article in. " +
							"Use twdesk-list_helpdoc_sites to discover.",
					},
					"title": {
						Type:        "string",
						Description: "The title of the article.",
					},
					"contents": {
						Description: "The body content of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "A short description / summary of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"status": {
						Description: "Publication status of the article (e.g. \"published\", \"draft\").",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"isPrivate": {
						Description: "Set to true to make the article private.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"siteID", "title", "contents", "description", "status", "isPrivate"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			title := arguments.GetString("title", "")
			article := deskmodels.HelpDocArticle{
				Helpdocsite: deskmodels.EntityRef{
					ID:   arguments.GetInt("siteID", 0),
					Type: "helpdocsites",
				},
				Title: &title,
			}

			contents := arguments.GetString("contents", "")
			if contents != "" {
				article.Contents = &contents
			}

			description := arguments.GetString("description", "")
			if description != "" {
				article.Description = &description
			}

			status := arguments.GetString("status", "")
			if status != "" {
				article.Status = &status
			}

			if arguments["isPrivate"] != nil {
				isPrivate := arguments.GetBool("isPrivate", false)
				article.IsPrivate = &isPrivate
			}

			result, err := client.HelpDocArticles.Create(ctx, &deskmodels.HelpDocArticleResponse{
				HelpDocArticle: article,
			})
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create help doc article")
			}
			return helpers.NewToolResultText("Help doc article created successfully with ID %d", result.HelpDocArticle.ID), nil
		},
	}
}

// HelpDocArticleUpdate updates an existing help doc article.
func HelpDocArticleUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocArticleUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Help Doc Article",
				// Changing status to published or toggling isPrivate alters
				// publicly-visible knowledge-base content.
				OpenWorldHint:   new(true),
				DestructiveHint: new(false),
			},
			Description: "Update an existing help doc article.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the help doc article to update.",
					},
					"title": {
						Description: "The new title of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"contents": {
						Description: "The new body content of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "A short description / summary of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"status": {
						Description: "Publication status (e.g. \"published\", \"draft\").",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"isPrivate": {
						Description: "Set to true to make the article private.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id", "title", "contents", "description", "status", "isPrivate"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			article := deskmodels.HelpDocArticle{}

			title := arguments.GetString("title", "")
			if title != "" {
				article.Title = &title
			}

			contents := arguments.GetString("contents", "")
			if contents != "" {
				article.Contents = &contents
			}

			description := arguments.GetString("description", "")
			if description != "" {
				article.Description = &description
			}

			status := arguments.GetString("status", "")
			if status != "" {
				article.Status = &status
			}

			if arguments["isPrivate"] != nil {
				isPrivate := arguments.GetBool("isPrivate", false)
				article.IsPrivate = &isPrivate
			}

			_, err = client.HelpDocArticles.Update(ctx, arguments.GetInt("id", 0), &deskmodels.HelpDocArticleResponse{
				HelpDocArticle: article,
			})
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update help doc article")
			}
			return helpers.NewToolResultText("Help doc article updated successfully"), nil
		},
	}
}
