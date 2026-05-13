package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomFieldCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldCreate.String(), map[string]any{
		"name":        "Priority Score",
		"type":        "number-integer",
		"entity":      "task",
		"description": "Priority score for tasks",
		"required":    true,
		"project_id":  float64(456),
	})
}

func TestCustomFieldCreateDropdown(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldCreate.String(), map[string]any{
		"name":   "Status",
		"type":   "dropdown",
		"entity": "task",
		"options": map[string]any{
			"choices": []any{
				map[string]any{"value": "Open", "color": "#ff0000"},
				map[string]any{"value": "Closed", "color": "#00ff00"},
			},
		},
	})
}

func TestCustomFieldUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Updated name",
		"description": "Updated description",
	})
}

func TestCustomFieldDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCustomFieldGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCustomFieldList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfields":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldList.String(), map[string]any{
		"search_term":     "priority",
		"ids":             []int64{1, 2, 3},
		"entities":        []string{"task", "project"},
		"project_ids":     []int64{10},
		"only_site_level": false,
		"order_by":        "name",
		"order_mode":      "asc",
		"page":            float64(1),
		"page_size":       float64(10),
	})
}
