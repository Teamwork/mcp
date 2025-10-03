//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestTypeCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket_type":{"id":123,"name":"Bug Report","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeCreate.String(), map[string]any{
		"name":  "Bug Report",
		"color": "red",
	})
}

func TestTypeUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_type":{"id":123,"name":"Feature Request","color":"blue"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeUpdate.String(), map[string]any{
		"id":    "123",
		"name":  "Feature Request",
		"color": "blue",
	})
}

func TestTypeGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_type":{"id":123,"name":"Support","color":"green"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeGet.String(), map[string]any{
		"id": "123",
	})
}

func TestTypeList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_types":[{"id":123,"name":"Bug Report","color":"red"},{"id":124,"name":"Feature Request","color":"blue"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeList.String(), map[string]any{
		"name":     []string{"Bug Report", "Feature Request"},
		"color":    []string{"red", "blue"},
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestTypeListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_types":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeList.String(), map[string]any{})
}
