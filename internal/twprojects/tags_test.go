package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTagCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"tag":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTagCreate.String(), map[string]any{
		"name":       "Example",
		"project_id": float64(456),
	})
}

func TestTagUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTagUpdate.String(), map[string]any{
		"id":         float64(123),
		"name":       "Example",
		"project_id": float64(456),
	})
}

func TestTagDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTagDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTagGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTagGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTagList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTagList.String(), map[string]any{
		"search_term": "test",
		"item_type":   "task",
		"project_ids": []int64{1, 2, 3},
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
