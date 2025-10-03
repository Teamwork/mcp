//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestUserGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"user":{"id":123,"firstName":"John","lastName":"Doe","email":"john@example.com"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodUserGet.String(), map[string]any{
		"id": "123",
	})
}

func TestUserList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"users":[{"id":123,"firstName":"John","lastName":"Doe"},{"id":124,"firstName":"Jane","lastName":"Smith"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodUserList.String(), map[string]any{
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestUserListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"users":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodUserList.String(), map[string]any{})
}
