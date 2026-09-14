//nolint:lll
package twdesk_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestHelpDocArticleGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpDocArticle":{"id":42,"title":"Getting Started","status":"published"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
		"id":     float64(42),
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
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"helpDocArticle":{"id":99,"title":"New Article","status":"draft"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID":      float64(1),
		"title":       "New Article",
		"contents":    "Article body here.",
		"description": "A short summary.",
		"status":      "draft",
		"isPrivate":   false,
	})
}

func TestHelpDocArticleCreateMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"helpDocArticle":{"id":100,"title":"Minimal Article"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleCreate.String(), map[string]any{
		"siteID":      float64(2),
		"title":       "Minimal Article",
		"contents":    nil,
		"description": nil,
		"status":      nil,
		"isPrivate":   nil,
	})
}

func TestHelpDocArticleUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpDocArticle":{"id":42,"title":"Updated Article","status":"published"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleUpdate.String(), map[string]any{
		"id":          float64(42),
		"title":       "Updated Article",
		"contents":    "Updated body.",
		"description": nil,
		"status":      "published",
		"isPrivate":   nil,
	})
}

func TestHelpDocArticleUpdateMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpDocArticle":{"id":42}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleUpdate.String(), map[string]any{
		"id":          float64(42),
		"title":       nil,
		"contents":    nil,
		"description": nil,
		"status":      nil,
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
			[]byte(`{"helpDocArticle":{"id":42,"title":"Getting Started","helpdocsite":{"id":7,"type":"helpdocsites"}}}`))
		defer cleanup()

		testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
			"id":     float64(42),
			"fields": nil,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				HelpDocArticle struct {
					Meta struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"helpDocArticle"`
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
			[]byte(`{"helpDocArticle":{"id":42,"title":"Orphan"}}`))
		defer cleanup()

		testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
			"id":     float64(42),
			"fields": nil,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				HelpDocArticle struct {
					Meta *struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"helpDocArticle"`
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
		[]byte(`{"helpDocArticle":{"id":42,"helpdocsite":{"id":7,"type":"helpdocsites"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocArticleGet.String(), map[string]any{
		"id":     float64(42),
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
