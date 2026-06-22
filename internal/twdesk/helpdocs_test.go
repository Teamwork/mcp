//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

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
