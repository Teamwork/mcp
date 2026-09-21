package twdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
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
	MethodHelpDocCategoryList  toolsets.Method = "twdesk-list_helpdoc_categories"
)

// helpDocArticleStatuses is the publication vocabulary the API accepts. It
// rejects anything else, and no response names the alternatives.
var helpDocArticleStatuses = []any{"Draft", "Unpublished", "Published"}

// helpDocArticleEditMethod is the only edit method these tools write.
//
// The create and update routes accept an article whose edit method is "html"
// and answer 400 "contents: must be set" for "markdown" or "htmlSource" —
// on create, and on update for an article already stored as either, down to a
// title-only change. Neither writes anything in that case. So the edit method
// is not a caller choice: publishing it as one would offer two values that
// always fail, and the error names the article's contents rather than the edit
// method, which reads as a missing body the caller did in fact send.
const helpDocArticleEditMethod = "html"

// helpDocArticleService addresses the help doc article routes of one site.
//
// The SDK's own HelpDocArticleService cannot reach them: it hangs every
// operation off "helpdocssites/helpdocarticles", where only the cross-site list
// lives. Get, create and update are each keyed on the site
// ("helpdocssites/{siteID}/helpdocarticles"), and update is a PATCH, so the
// SDK's site-less PUT answers 404 as well.
//
// Both type parameters are map[string]any rather than the SDK's
// HelpDocArticleResponse, which spells the wrapper key "helpDocArticle" where
// every route reads and answers "helpdocarticle" — a response decodes into an
// empty struct and a write body binds nothing. A map also carries a sparse
// selection back untouched, and lets a write send only the keys the caller
// named: the update route binds the body over the stored article and
// revalidates the result, so a key sent as its zero value is an erasure
// ("categories": null clears them, and the revalidation then rejects the update
// for having none).
func helpDocArticleService(
	client *deskclient.Client,
	siteID int,
) *deskclient.Service[map[string]any, map[string]any] {
	return deskclient.NewService[map[string]any, map[string]any](client,
		deskclient.NewDefaultPathHandlerWithUpdateMethod(
			fmt.Sprintf("helpdocssites/%d/helpdocarticles", siteID), http.MethodPatch,
		),
	)
}

// helpDocCategoryService addresses the help doc category routes, which the SDK
// models no service for at all.
func helpDocCategoryService(client *deskclient.Client) *deskclient.Service[map[string]any, map[string]any] {
	return deskclient.NewService[map[string]any, map[string]any](client,
		deskclient.NewDefaultPathHandler("helpdocscategories"),
	)
}

// helpDocArticleBody wraps an article under the key the routes read and answer.
func helpDocArticleBody(article map[string]any) map[string]any {
	return map[string]any{"helpdocarticle": article}
}

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
			Description: "Get a help doc article by ID. The article is addressed through its site, " +
				"so both IDs are needed; twdesk-search_helpdoc_articles reports each article's site.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the help doc article to retrieve.",
					},
					"siteID": {
						Type: "integer",
						Description: "The ID of the help doc site the article belongs to. " +
							"Use twdesk-list_helpdoc_sites to discover.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id", "siteID", "fields"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			service := helpDocArticleService(client, arguments.GetInt("siteID", 0))
			article, err := service.Get(ctx, arguments.GetInt("id", 0), helpDocArticleGetParams(arguments))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get help doc article")
			}
			return helpDocArticleResult(ctx, *article)
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
			return helpDocArticleResult(ctx, articles)
		},
	}
}

// helpDocArticleResult answers with the article payload plus a meta.webLink on
// every article it carries, on both the text and the structured content.
func helpDocArticleResult(ctx context.Context, payload any) (*mcp.CallToolResult, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return helpers.NewToolResultTextError("failed to encode help doc article: %s", err.Error()), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(helpers.WebLinker(ctx, encoded, helpDocArticlePathBuilder)),
			},
		},
		StructuredContent: helpers.StructuredWebLinker(ctx, payload, helpDocArticlePathBuilder),
	}, nil
}

// helpDocArticlePathBuilder builds the Desk path for a help doc article. The
// route is keyed on the help doc site as well as the article, so
// helpers.WebLinkerWithIDPathBuilder cannot serve it. An article carrying no
// site ID gets no link rather than one pointing at the wrong site.
//
// The path stops at the article: it is the parent route, which redirects to
// whichever tab the app treats as the default (today "edit", beside "related",
// "keywords" and "revisions"). Naming the tab here would pin a choice that is
// the app's to make.
func helpDocArticlePathBuilder(object map[string]any) string {
	articleID, ok := helpDocIdentifier(object["id"])
	if !ok {
		return ""
	}
	site, ok := object["helpdocsite"].(map[string]any)
	if !ok {
		return ""
	}
	siteID, ok := helpDocIdentifier(site["id"])
	if !ok {
		return ""
	}
	return fmt.Sprintf("/desk/help-docs/%d/article/%d", siteID, articleID)
}

// helpDocIdentifier reads a JSON-decoded identifier as a positive integer. The
// linker hands over decoded JSON, so an ID arrives as a float64 and would
// otherwise render with a decimal point.
func helpDocIdentifier(value any) (int64, bool) {
	var id int64
	switch v := value.(type) {
	case float64:
		if math.Trunc(v) != v {
			return 0, false
		}
		id = int64(v)
	case int:
		id = int64(v)
	case int64:
		id = v
	default:
		return 0, false
	}
	if id <= 0 {
		return 0, false
	}
	return id, true
}

// helpDocArticleGetParams is getParams with the two attributes the web link is
// built from appended to any sparse selection: the article's own ID, and the
// site the Desk route is keyed on. A caller naming fields has no way to know
// the link depends on them, and a selection that leaves them out would answer
// with no link at all.
func helpDocArticleGetParams(arguments helpers.ToolArguments) url.Values {
	params := getParams(arguments)

	fields := arguments.GetStringSlice("fields", nil)
	if len(fields) == 0 {
		return params
	}
	for _, needed := range []string{"id", "helpdocsite"} {
		if !slices.Contains(fields, needed) {
			fields = append(fields, needed)
		}
	}
	params.Set("fields", strings.Join(fields, ","))
	return params
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
			Description: "Create a new help doc article, with an HTML body. An article always " +
				"belongs to at least one category of its site, so a category ID is required: list " +
				"them with twdesk-list_helpdoc_categories. The article is stored as HTML whatever " +
				"the site's own editing mode is, and articles already stored as Markdown cannot be " +
				"created or edited through this API at all.",
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
						Description: "The title of the article (1-255 characters).",
					},
					"contents": {
						Type: "string",
						Description: "The body of the article, as HTML. The endpoint stores help doc " +
							"articles written through this tool as HTML, so Markdown arrives as literal " +
							"text rather than being rendered.",
					},
					"categoryIDs": {
						Type:  "array",
						Items: &jsonschema.Schema{Type: "integer"},
						Description: "The categories to file the article under. At least one is required, " +
							"and each must belong to the same site. Use twdesk-list_helpdoc_categories.",
					},
					"description": {
						Description: "A short description / summary of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"status": {
						Description: "Publication status of the article. Defaults to Draft.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: helpDocArticleStatuses},
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
				Required: []string{
					"siteID", "title", "contents", "categoryIDs",
					"description", "status", "isPrivate",
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			siteID := arguments.GetInt("siteID", 0)

			categoryIDs := arguments.GetIntSlice("categoryIDs", nil)
			if len(categoryIDs) == 0 {
				return helpers.NewToolResultTextError("categoryIDs is required: an article must be filed "+
					"under at least one category of site %d, which twdesk-list_helpdoc_categories lists",
					siteID), nil
			}

			// editMethod is sent rather than taken from the caller or from the
			// site: the route refuses every other value. Inheriting the site's
			// setting was the obvious reading and is wrong — a site whose own mode
			// is Markdown then fails every create.
			article := map[string]any{
				"title":      arguments.GetString("title", ""),
				"contents":   arguments.GetString("contents", ""),
				"categories": categoryIDs,
				"editMethod": helpDocArticleEditMethod,
				"status":     arguments.GetString("status", "Draft"),
			}
			if description := arguments.GetString("description", ""); description != "" {
				article["description"] = description
			}
			if arguments["isPrivate"] != nil {
				article["isPrivate"] = arguments.GetBool("isPrivate", false)
			}

			body := helpDocArticleBody(article)
			created, err := helpDocArticleService(client, siteID).Create(ctx, &body)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create help doc article")
			}
			return helpers.NewToolResultText("Help doc article created successfully with ID %d",
				helpDocArticleID(*created)), nil
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
			Description: "Update an existing help doc article. Only the properties supplied " +
				"change; the article is addressed through its site, so both IDs are needed. An " +
				"article stored as Markdown cannot be updated at all — the endpoint answers 400 " +
				"\"contents: must be set\" even for a title-only change, and writes nothing. Read " +
				"the article's editMethod first if it matters.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the help doc article to update.",
					},
					"siteID": {
						Type: "integer",
						Description: "The ID of the help doc site the article belongs to. " +
							"Use twdesk-list_helpdoc_sites to discover.",
					},
					"title": {
						Description: "The new title of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"contents": {
						Description: "The new body of the article, as HTML.",
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
						Description: "Publication status of the article.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: helpDocArticleStatuses},
							{Type: "null"},
						},
					},
					"categoryIDs": {
						Description: "Replace the categories the article is filed under. Every ID must " +
							"belong to the article's site, and the list cannot be emptied.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
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
				Required: []string{
					"id", "siteID", "title", "contents",
					"description", "status", "categoryIDs", "isPrivate",
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Only what the caller named is sent: the endpoint binds the body over
			// the stored article, so an unmentioned property left in the body as its
			// zero value would overwrite what is there.
			article := map[string]any{}
			if title := arguments.GetString("title", ""); title != "" {
				article["title"] = title
			}
			if contents := arguments.GetString("contents", ""); contents != "" {
				article["contents"] = contents
			}
			if description := arguments.GetString("description", ""); description != "" {
				article["description"] = description
			}
			if status := arguments.GetString("status", ""); status != "" {
				article["status"] = status
			}
			if categoryIDs := arguments.GetIntSlice("categoryIDs", nil); len(categoryIDs) > 0 {
				article["categories"] = categoryIDs
			}
			if arguments["isPrivate"] != nil {
				article["isPrivate"] = arguments.GetBool("isPrivate", false)
			}

			body := helpDocArticleBody(article)
			service := helpDocArticleService(client, arguments.GetInt("siteID", 0))
			if _, err := service.Update(ctx, arguments.GetInt("id", 0), &body); err != nil {
				return helpers.HandleAPIError(err, "failed to update help doc article")
			}
			return helpers.NewToolResultText("Help doc article updated successfully"), nil
		},
	}
}

// HelpDocCategoryList lists the categories of a help doc site.
func HelpDocCategoryList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocCategoryList),
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Help Doc Categories",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "List help doc categories. Every article is filed under at least one, so " +
				"twdesk-create_helpdoc_article needs an ID from here.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: paginationOptions(map[string]*jsonschema.Schema{
					"siteID": {
						Description: "Filter by help doc site ID. Omit to list the categories of every " +
							"site. Use twdesk-list_helpdoc_sites to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				}),
				Required: append([]string{"siteID"}, paginationRequiredKeys()...),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			params := url.Values{}
			setPagination(&params, arguments)

			if siteID := arguments.GetInt("siteID", 0); siteID > 0 {
				filter, err := json.Marshal(map[string]any{"sites.id": siteID})
				if err != nil {
					return helpers.NewToolResultTextError("failed to encode the site filter: %s", err.Error()), nil
				}
				params.Set("filter", string(filter))
			}

			categories, err := helpDocCategoryService(client).List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list help doc categories")
			}
			return helpers.NewToolResultJSON(*categories)
		},
	}
}

// helpDocArticleID reads the ID out of a response body, which carries the
// article under the key the routes answer with.
func helpDocArticleID(body map[string]any) int64 {
	article, ok := body["helpdocarticle"].(map[string]any)
	if !ok {
		return 0
	}
	id, _ := helpDocIdentifier(article["id"])
	return id
}
