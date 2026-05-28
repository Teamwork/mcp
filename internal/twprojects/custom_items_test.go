package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomItemCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customItem":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemCreate.String(), map[string]any{
		"project_id":     float64(456),
		"display_name":   "Contracts",
		"description":    "Customer contracts",
		"label_singular": "Contract",
		"label_plural":   "Contracts",
	})
}

func TestCustomItemUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItem":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemUpdate.String(), map[string]any{
		"id":             float64(123),
		"display_name":   "Updated Contracts",
		"description":    "Updated description",
		"label_singular": "Contract",
		"label_plural":   "Contracts",
	})
}

func TestCustomItemDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCustomItemGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItem":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCustomItemList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItems":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemList.String(), map[string]any{
		"project_id":   float64(456),
		"search_term":  "contract",
		"ids":          []int64{1, 2, 3},
		"show_deleted": false,
		"order_by":     "name",
		"order_mode":   "asc",
		"page":         float64(1),
		"page_size":    float64(10),
	})
}
