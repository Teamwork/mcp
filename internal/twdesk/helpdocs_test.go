//nolint:lll
package twdesk_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestHelpDocArticleGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocarticle":{"id":42,"title":"Getting Started","status":"published"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
		"id":     float64(42),
		"siteID": float64(7),
		"fields": nil,
	})
}

func TestHelpDocArticleSearch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocarticles":[{"id":42,"title":"Getting Started","status":"published"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleSearch.String(), map[string]any{
		"search":     "Getting Started",
		"status":     "published",
		"siteID":     float64(1),
		"categoryID": nil,
		"page":       float64(1),
		"pageSize":   float64(10),
	})
}

func TestHelpDocArticleSearchMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocarticles":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleSearch.String(), map[string]any{
		"search":     nil,
		"status":     nil,
		"siteID":     nil,
		"categoryID": nil,
		"page":       nil,
		"pageSize":   nil,
	})
}

func TestHelpDocArticleCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"helpdocarticle":{"id":99,"title":"New Article","status":"draft"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID":      float64(1),
		"title":       "New Article",
		"contents":    "<p>Article body here.</p>",
		"categoryIDs": []any{float64(5)},
		"description": "A short summary.",
		"status":      "Draft",
		"isPrivate":   false,
	})
}

func TestHelpDocArticleCreateMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"helpdocarticle":{"id":100,"title":"Minimal Article"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID":      float64(2),
		"title":       "Minimal Article",
		"contents":    "Article body here.",
		"categoryIDs": []any{float64(5)},
		"description": nil,
		"status":      nil,
		"isPrivate":   nil,
	})
}

func TestHelpDocArticleUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocarticle":{"id":42,"title":"Updated Article","status":"published"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleUpdate.String(), map[string]any{
		"id":          float64(42),
		"siteID":      float64(7),
		"title":       "Updated Article",
		"contents":    "Updated body.",
		"description": nil,
		"status":      "Published",
		"categoryIDs": nil,
		"isPrivate":   nil,
	})
}

func TestHelpDocArticleUpdateMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocarticle":{"id":42}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleUpdate.String(), map[string]any{
		"id":          float64(42),
		"siteID":      float64(7),
		"title":       nil,
		"contents":    nil,
		"description": nil,
		"status":      nil,
		"categoryIDs": nil,
		"isPrivate":   nil,
	})
}

// TestHelpDocArticleWebLink pins the meta.webLink both article tools attach.
//
// The link is keyed on the help doc site as well as the article, so it cannot
// come from helpers.WebLinkerWithIDPathBuilder and nothing else in the response
// reveals whether it was built from the right pair of IDs.
func TestHelpDocArticleWebLink(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		mcpServer, customerURL, cleanup := testutil.DeskMCPServerMockWithCustomerURL(t, http.StatusOK,
			[]byte(`{"helpdocarticle":{"id":42,"title":"Getting Started","helpdocsite":{"id":7,"type":"helpdocsites"}}}`))
		defer cleanup()

		testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
			"id":     float64(42),
			"siteID": float64(7),
			"fields": nil,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				HelpDocArticle struct {
					Meta struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"helpdocarticle"`
			}
			decodeToolResult(t, result, &payload)

			if want := customerURL + "/desk/help-docs/7/article/42"; payload.HelpDocArticle.Meta.WebLink != want {
				t.Errorf("expected web link %q, got %q", want, payload.HelpDocArticle.Meta.WebLink)
			}
		}))
	})

	t.Run("search", func(t *testing.T) {
		mcpServer, customerURL, cleanup := testutil.DeskMCPServerMockWithCustomerURL(t, http.StatusOK,
			[]byte(`{"helpdocarticles":[`+
				`{"id":42,"title":"Getting Started","helpdocsite":{"id":7,"type":"helpdocsites"}},`+
				`{"id":43,"title":"Next Steps","helpdocsite":{"id":8,"type":"helpdocsites"}}]}`))
		defer cleanup()

		testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleSearch.String(), map[string]any{
			"search":     "Getting Started",
			"status":     nil,
			"siteID":     nil,
			"categoryID": nil,
			"page":       nil,
			"pageSize":   nil,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				HelpDocArticles []struct {
					Meta struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"helpdocarticles"`
			}
			decodeToolResult(t, result, &payload)

			if len(payload.HelpDocArticles) != 2 {
				t.Fatalf("expected 2 articles, got %d", len(payload.HelpDocArticles))
			}
			for i, want := range []string{
				customerURL + "/desk/help-docs/7/article/42",
				customerURL + "/desk/help-docs/8/article/43",
			} {
				if got := payload.HelpDocArticles[i].Meta.WebLink; got != want {
					t.Errorf("article %d: expected web link %q, got %q", i, want, got)
				}
			}
		}))
	})

	// An article with no site cannot be addressed by the Desk route, and a link
	// built from the article ID alone would point at another site's article.
	t.Run("no site ID", func(t *testing.T) {
		mcpServer, _, cleanup := testutil.DeskMCPServerMockWithCustomerURL(t, http.StatusOK,
			[]byte(`{"helpdocarticle":{"id":42,"title":"Orphan"}}`))
		defer cleanup()

		testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
			"id":     float64(42),
			"siteID": float64(7),
			"fields": nil,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				HelpDocArticle struct {
					Meta *struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"helpdocarticle"`
			}
			decodeToolResult(t, result, &payload)

			if payload.HelpDocArticle.Meta != nil {
				t.Errorf("expected no web link, got %q", payload.HelpDocArticle.Meta.WebLink)
			}
		}))
	})
}

// TestHelpDocArticleGetFieldsCarryTheWebLinkAttributes asserts that a sparse
// selection still asks for the attributes the link is built from. Without them
// the endpoint answers a record the linker cannot address, and the caller has
// no way to know the link depends on fields it did not name.
func TestHelpDocArticleGetFieldsCarryTheWebLinkAttributes(t *testing.T) {
	mcpServer, requestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"helpdocarticle":{"id":42,"helpdocsite":{"id":7,"type":"helpdocsites"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
		"id":     float64(42),
		"siteID": float64(7),
		"fields": []any{"title"},
	})

	sent := requestURL()
	if got, want := sent.Query().Get("fields"), "title,id,helpdocsite"; got != want {
		t.Errorf("expected fields %q, got %q", want, got)
	}
}

// decodeToolResult decodes the tool's text content, which is the payload a
// client that ignores structured content reads.
func decodeToolResult(t *testing.T, result mcp.Result, target any) {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if len(toolResult.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(toolResult.Content))
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), target); err != nil {
		t.Fatalf("failed to decode tool result %q: %v", text.Text, err)
	}
}

// TestHelpDocArticleRoutesAreSiteScoped pins the route each article tool
// addresses, and the verb it uses.
//
// The article routes hang off the site — helpdocssites/{siteID}/helpdocarticles
// — and update is a PATCH. The SDK's own service addresses them without the
// site and updates with a PUT, so every one of these calls answered 404 while
// the reads that do not go through it (search, sites) kept working, which is
// what made this look like a permissions problem rather than a routing one.
func TestHelpDocArticleRoutesAreSiteScoped(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		args       map[string]any
		wantMethod string
		wantPath   string
	}{{
		name:   "get",
		method: twdesk.MethodHelpDocArticleGet.String(),
		args: map[string]any{
			"id": float64(42), "siteID": float64(7), "fields": nil,
		},
		wantMethod: http.MethodGet,
		wantPath:   "/desk/api/v2/helpdocssites/7/helpdocarticles/42.json",
	}, {
		name:   "create",
		method: twdesk.MethodHelpDocArticleCreate.String(),
		args: map[string]any{
			"siteID": float64(7), "title": "New Article", "contents": "Body.",
			"categoryIDs": []any{float64(5)}, "description": nil, "status": nil, "isPrivate": nil,
		},
		wantMethod: http.MethodPost,
		wantPath:   "/desk/api/v2/helpdocssites/7/helpdocarticles.json",
	}, {
		name:   "update",
		method: twdesk.MethodHelpDocArticleUpdate.String(),
		args: map[string]any{
			"id": float64(42), "siteID": float64(7), "title": "Updated", "contents": nil,
			"description": nil, "status": nil, "categoryIDs": nil, "isPrivate": nil,
		},
		wantMethod: http.MethodPatch,
		wantPath:   "/desk/api/v2/helpdocssites/7/helpdocarticles/42.json",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequest, cleanup := testutil.DeskMCPServerMockWithRequest(t, http.StatusOK,
				[]byte(`{"helpdocarticle":{"id":42}}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.args)

			method, requestURL := lastRequest()
			if method != tt.wantMethod {
				t.Errorf("expected method %s, got %s", tt.wantMethod, method)
			}
			if requestURL.Path != tt.wantPath {
				t.Errorf("expected path %q, got %q", tt.wantPath, requestURL.Path)
			}
		})
	}
}

// TestHelpDocArticleWriteBodyCarriesTheAPIWrapperKey pins the key a write body
// is wrapped in, and the properties underneath it.
//
// The SDK's HelpDocArticleResponse spells it "helpDocArticle", which no route
// reads: the endpoint binds nothing and rejects the create as having no title.
// Nothing in a response shows the difference, since the same canned body comes
// back either way.
func TestHelpDocArticleWriteBodyCarriesTheAPIWrapperKey(t *testing.T) {
	mcpServer, lastRequest, cleanup := testutil.DeskMCPServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"helpdocarticle":{"id":99}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID": float64(7), "title": "New Article", "contents": "Body.",
		"categoryIDs": []any{float64(5), float64(6)}, "description": "Summary.",
		"status": "Published", "isPrivate": true,
	})

	_, _, body := lastRequest()

	var sent struct {
		HelpDocArticle map[string]any `json:"helpdocarticle"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("failed to decode request body %q: %v", body, err)
	}
	if sent.HelpDocArticle == nil {
		t.Fatalf("expected the article under %q, got %q", "helpdocarticle", body)
	}
	for key, want := range map[string]any{
		"title":       "New Article",
		"contents":    "Body.",
		"description": "Summary.",
		"status":      "Published",
		"isPrivate":   true,
		"categories":  []any{float64(5), float64(6)},
	} {
		if got := sent.HelpDocArticle[key]; !reflect.DeepEqual(got, want) {
			t.Errorf("expected %s %v, got %v", key, want, got)
		}
	}
}

// TestHelpDocArticleCreateNeedsACategory pins that the create tool rejects an
// empty category list itself.
//
// The API requires at least one, and answers a create without one with a
// validation error naming a property the tool did not advertise.
func TestHelpDocArticleCreateNeedsACategory(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"helpdocarticle":{"id":99}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID": float64(7), "title": "New Article", "contents": "Body.",
		"categoryIDs": []any{}, "description": nil, "status": nil, "isPrivate": nil,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Error("expected an error result for a create carrying no category")
		}
	}))
}

// TestHelpDocArticleCreateAlwaysSendsHTML pins the edit method the create tool
// writes, and that it asks the site for nothing.
//
// The create route accepts an article whose edit method is "html" and answers
// 400 "contents: must be set" for "markdown" or "htmlSource", naming the body
// the caller did send rather than the setting it rejected. Inheriting the
// site's own mode is the obvious reading and fails every create on a Markdown
// site, so the value is fixed here and the tool advertises no parameter for it.
func TestHelpDocArticleCreateAlwaysSendsHTML(t *testing.T) {
	mcpServer, lastRequest, cleanup := testutil.DeskMCPServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"helpdocarticle":{"id":99}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID": float64(7), "title": "New Article", "contents": "<p>Body.</p>",
		"categoryIDs": []any{float64(5)}, "description": nil, "status": nil, "isPrivate": nil,
	})

	method, requestURL, body := lastRequest()

	// One request: the create. A site read to discover the edit method would
	// make this the second, and the last request would not be the POST.
	if method != http.MethodPost {
		t.Errorf("expected the last request to be the create POST, got %s %s", method, requestURL.Path)
	}

	var sent struct {
		HelpDocArticle struct {
			EditMethod string `json:"editMethod"`
			Status     string `json:"status"`
		} `json:"helpdocarticle"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("failed to decode request body %q: %v", body, err)
	}
	if got, want := sent.HelpDocArticle.EditMethod, "html"; got != want {
		t.Errorf("expected edit method %q, got %q", want, got)
	}
	if got, want := sent.HelpDocArticle.Status, "Draft"; got != want {
		t.Errorf("expected status %q, got %q", want, got)
	}
}

// TestHelpDocArticleCreatePublishesNoEditMethod keeps the parameter off the
// published schema. Two of the three values the API models always fail, and a
// caller cannot tell that from the error, so offering the choice costs a failed
// call to discover.
func TestHelpDocArticleCreatePublishesNoEditMethod(t *testing.T) {
	group := twdesk.DefaultToolsetGroup(false, nil)
	for _, toolset := range group.Toolsets {
		for _, wrapper := range toolset.GetAvailableTools() {
			if wrapper.Tool.Name != twdesk.MethodHelpDocArticleCreate.String() {
				continue
			}
			schema, ok := wrapper.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				t.Fatalf("InputSchema is not *jsonschema.Schema (got %T)", wrapper.Tool.InputSchema)
			}
			if _, ok := schema.Properties["editMethod"]; ok {
				t.Error("create should not advertise editMethod: only one of its values is accepted")
			}
			return
		}
	}
	t.Fatal("create tool not found")
}

// TestHelpDocArticleUpdateSendsOnlyWhatTheCallerNamed pins that an update
// carries no property the caller left out.
//
// The update route loads the stored article, binds the body over it and
// revalidates the result, so a property sent as its zero value is an erasure: a
// "categories": null clears them and the revalidation then rejects the update
// for having none, which is how the SDK's own model — whose categories field
// carries no omitempty — would fail every update.
func TestHelpDocArticleUpdateSendsOnlyWhatTheCallerNamed(t *testing.T) {
	mcpServer, lastRequest, cleanup := testutil.DeskMCPServerMockWithRequestBody(t, http.StatusOK,
		[]byte(`{"helpdocarticle":{"id":42}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleUpdate.String(), map[string]any{
		"id": float64(42), "siteID": float64(7), "title": "Updated Article",
		"contents": nil, "description": nil, "status": nil, "categoryIDs": nil, "isPrivate": nil,
	})

	_, _, body := lastRequest()

	var sent struct {
		HelpDocArticle map[string]any `json:"helpdocarticle"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("failed to decode request body %q: %v", body, err)
	}
	if got, want := sent.HelpDocArticle["title"], "Updated Article"; got != want {
		t.Errorf("expected title %q, got %v", want, got)
	}
	for _, key := range []string{"categories", "contents", "description", "status", "isPrivate"} {
		if _, ok := sent.HelpDocArticle[key]; ok {
			t.Errorf("expected no %s in the body, got %v", key, sent.HelpDocArticle[key])
		}
	}
}

// TestHelpDocCategoryListFiltersBySite pins the query string the category list
// builds, since the mock answers with the same body whether the filter is sent
// or not.
func TestHelpDocCategoryListFiltersBySite(t *testing.T) {
	mcpServer, requestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"helpdocscategories":[{"id":5,"name":"Getting Started"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocCategoryList.String(), map[string]any{
		"siteID": float64(7), "page": nil, "pageSize": nil,
		"orderBy": nil, "orderDirection": nil, "fields": nil,
	})

	sent := requestURL()
	if got, want := sent.Path, "/desk/api/v2/helpdocscategories.json"; got != want {
		t.Errorf("expected path %q, got %q", want, got)
	}
	if got, want := sent.Query().Get("filter"), `{"sites.id":7}`; got != want {
		t.Errorf("expected filter %q, got %q", want, got)
	}
}
