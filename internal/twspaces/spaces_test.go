//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestSpaceGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"space":{"id":1,"title":"Engineering","code":"ENG","state":"active","spaceColor":"#FF5733","icon":"rocket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceGet.String(), map[string]any{
		"id": float64(1),
	})
}

func TestSpaceList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"spaces":[{"id":1,"title":"Engineering","code":"ENG","state":"active","spaceColor":"#FF5733","icon":"rocket"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceList.String(), map[string]any{})
}

func TestSpaceCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"space":{"id":1,"title":"Engineering","code":"ENG","state":"active","spaceColor":"#FF5733","icon":"rocket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceCreate.String(), map[string]any{
		"title": "Engineering",
		"code":  "ENG",
	})
}

func TestSpaceCreateWithOptionals(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"space":{"id":1,"title":"Engineering","code":"ENG","state":"active","spaceColor":"#FF5733","icon":"rocket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceCreate.String(), map[string]any{
		"title":      "Engineering",
		"code":       "ENG",
		"purpose":    "Team documentation",
		"spaceColor": "#FF5733",
		"icon":       "rocket",
		"projectId":  float64(42),
		"categoryId": float64(3),
	})
}

func TestSpaceUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"space":{"id":1,"title":"Engineering Updated","code":"ENG","state":"active","spaceColor":"#FF5733","icon":"rocket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceUpdate.String(), map[string]any{
		"id":    float64(1),
		"title": "Engineering Updated",
	})
}

func TestSpaceDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceDelete.String(), map[string]any{
		"id": float64(1),
	})
}

func TestSpaceCollaborators(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"collaborators":[{"id":1,"type":"user"},{"id":2,"type":"user"}],"collaboratorsCount":2}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSpaceCollaborators.String(), map[string]any{
		"id": float64(1),
	})
}
