//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestTagCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"tag":{"id":123,"name":"urgent","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagCreate.String(), map[string]any{
		"name":  "urgent",
		"color": "red",
	})
}

func TestTagUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tag":{"id":123,"name":"important","color":"orange"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagUpdate.String(), map[string]any{
		"id":    "123",
		"name":  "important",
		"color": "orange",
	})
}

func TestTagGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tag":{"id":123,"name":"urgent","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagGet.String(), map[string]any{
		"id": "123",
	})
}

func TestTagList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[{"id":123,"name":"urgent","color":"red"},{"id":124,"name":"important","color":"orange"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagList.String(), map[string]any{
		"name":     []string{"urgent", "important"},
		"color":    []string{"red", "orange"},
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestTagListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagList.String(), map[string]any{})
}
