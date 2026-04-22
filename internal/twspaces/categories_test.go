//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestCategoryGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"category":{"id":1,"name":"Engineering","color":"#4287f5","meta":{"spaceCount":3}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryGet.String(), map[string]any{
		"id": float64(1),
	})
}

func TestCategoryList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"categories":[{"id":1,"name":"Engineering","color":"#4287f5","meta":{"spaceCount":3}},{"id":2,"name":"Marketing","color":"#f5a442","meta":{"spaceCount":1}}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryList.String(), map[string]any{})
}

func TestCategoryCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"category":{"id":1,"name":"Engineering","color":"#4287f5","meta":{"spaceCount":0}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryCreate.String(), map[string]any{
		"name":  "Engineering",
		"color": "#4287f5",
	})
}

func TestCategoryCreateMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"category":{"id":1,"name":"Engineering","meta":{"spaceCount":0}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryCreate.String(), map[string]any{
		"name": "Engineering",
	})
}

func TestCategoryUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"category":{"id":1,"name":"Platform Engineering","color":"#4287f5","meta":{"spaceCount":3}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryUpdate.String(), map[string]any{
		"id":   float64(1),
		"name": "Platform Engineering",
	})
}

func TestCategoryDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCategoryDelete.String(), map[string]any{
		"id": float64(1),
	})
}
