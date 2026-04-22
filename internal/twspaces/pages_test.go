//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestPageGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Getting Started","slug":"getting-started","content":"<p>Welcome</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageGet.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
	})
}

func TestPageList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"pages":{"id":0,"slug":"","title":"root","childPages":[{"id":10,"slug":"getting-started","title":"Getting Started","childPages":[]}]}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId": float64(1),
	})
}

func TestPageHome(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":1,"title":"Home","slug":"home","content":"<p>Welcome</p>","isHomePage":true,"state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageHome.String(), map[string]any{
		"spaceId": float64(1),
	})
}

func TestPageCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"New Page","slug":"new-page","content":"<p>Hello</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageCreate.String(), map[string]any{
		"spaceId": float64(1),
		"title":   "New Page",
		"content": "<p>Hello</p>",
	})
}

func TestPageCreateWithOptionals(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"New Page","slug":"new-page","content":"<p>Hello</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageCreate.String(), map[string]any{
		"spaceId":   float64(1),
		"title":     "New Page",
		"content":   "<p>Hello</p>",
		"parentId":  float64(5),
		"slug":      "new-page",
		"isPublish": true,
	})
}

func TestPageDuplicate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":11,"title":"Copy of Getting Started","slug":"copy-of-getting-started","content":"<p>Welcome</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageDuplicate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"title":   "Copy of Getting Started",
	})
}

func TestPageUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Updated Page","slug":"getting-started","content":"<p>Updated</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageUpdate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"title":   "Updated Page",
		"content": "<p>Updated</p>",
	})
}

func TestPageDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageDelete.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
	})
}
