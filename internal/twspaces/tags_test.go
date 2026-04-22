//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestTagGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":{"id":1,"name":"important","color":"#FF0000"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodTagGet.String(), map[string]any{
		"id": float64(1),
	})
}

func TestTagList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[{"id":1,"name":"important","color":"#FF0000"},{"id":2,"name":"draft","color":"#FFA500"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodTagList.String(), map[string]any{})
}

func TestTagCreateBatch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[{"id":1,"name":"important","color":"#FF0000"},{"id":2,"name":"draft","color":"#FFA500"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodTagCreateBatch.String(), map[string]any{
		"tags": []any{
			map[string]any{"name": "important", "color": "#FF0000"},
			map[string]any{"name": "draft", "color": "#FFA500"},
		},
	})
}

func TestTagUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":{"id":1,"name":"critical","color":"#FF0000"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodTagUpdate.String(), map[string]any{
		"id":   float64(1),
		"name": "critical",
	})
}

func TestTagDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodTagDelete.String(), map[string]any{
		"id": float64(1),
	})
}
