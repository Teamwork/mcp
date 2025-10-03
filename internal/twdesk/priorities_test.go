//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestPriorityCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket_priority":{"id":123,"name":"High","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodPriorityCreate.String(), map[string]any{
		"name":  "High",
		"color": "red",
	})
}

func TestPriorityUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_priority":{"id":123,"name":"Updated","color":"blue"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodPriorityUpdate.String(), map[string]any{
		"id":    "123",
		"name":  "Updated",
		"color": "blue",
	})
}

func TestPriorityGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_priority":{"id":123,"name":"Medium","color":"yellow"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodPriorityGet.String(), map[string]any{
		"id": "123",
	})
}

func TestPriorityList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_priorities":[{"id":123,"name":"High","color":"red"},{"id":124,"name":"Medium","color":"yellow"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodPriorityList.String(), map[string]any{
		"name":     []string{"High", "Medium"},
		"color":    []string{"red", "yellow"},
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestPriorityListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_priorities":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodPriorityList.String(), map[string]any{})
}
