package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomItemFieldCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customItemField":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldCreate.String(), map[string]any{
		"custom_item_id": float64(456),
		"display_name":   "Priority",
		"type":           "number-integer",
	})
}

func TestCustomItemFieldCreateDropdown(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customItemField":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldCreate.String(), map[string]any{
		"custom_item_id": float64(456),
		"display_name":   "Status",
		"type":           "dropdown",
		"tw_type":        "status",
		"options": []any{
			map[string]any{"label": "Open", "color": "#ff0000"},
			map[string]any{"label": "Closed", "color": "#00ff00"},
		},
		"definition": map[string]any{
			"required": true,
		},
		"position_after_id": float64(789),
	})
}

func TestCustomItemFieldUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItemField":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldUpdate.String(), map[string]any{
		"custom_item_id":    float64(456),
		"id":                float64(123),
		"display_name":      "Updated Priority",
		"definition":        map[string]any{"required": false},
		"position_after_id": float64(789),
	})
}

func TestCustomItemFieldDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldDelete.String(), map[string]any{
		"custom_item_id": float64(456),
		"id":             float64(123),
	})
}

func TestCustomItemFieldGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItemField":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldGet.String(), map[string]any{
		"custom_item_id": float64(456),
		"id":             float64(123),
	})
}

func TestCustomItemFieldList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customItemFields":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemFieldList.String(), map[string]any{
		"custom_item_id": float64(456),
		"search_term":    "priority",
		"ids":            []int64{1, 2, 3},
		"show_deleted":   false,
		"order_mode":     "asc",
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
