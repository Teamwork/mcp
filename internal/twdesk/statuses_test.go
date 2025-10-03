//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestStatusCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket_status":{"id":123,"name":"In Progress","color":"blue","displayOrder":1}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodStatusCreate.String(), map[string]any{
		"name":         "In Progress",
		"color":        "blue",
		"displayOrder": float64(1),
	})
}

func TestStatusUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_status":{"id":123,"name":"Completed","color":"green","displayOrder":2}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodStatusUpdate.String(), map[string]any{
		"id":           "123",
		"name":         "Completed",
		"color":        "green",
		"displayOrder": float64(2),
	})
}

func TestStatusGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_status":{"id":123,"name":"Open","color":"red","displayOrder":0}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodStatusGet.String(), map[string]any{
		"id": "123",
	})
}

func TestStatusList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_statuses":[{"id":123,"name":"Open","color":"red"},{"id":124,"name":"In Progress","color":"blue"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodStatusList.String(), map[string]any{
		"name":     []string{"Open", "In Progress"},
		"color":    []string{"red", "blue"},
		"code":     []string{"open", "in_progress"},
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestStatusListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_statuses":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodStatusList.String(), map[string]any{})
}
