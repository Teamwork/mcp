package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTasklistCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"tasklistId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistCreate.String(), map[string]any{
		"name":         "Example",
		"description":  "This is an example tasklist.",
		"project_id":   float64(456),
		"milestone_id": float64(789),
	})
}

func TestTasklistUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistUpdate.String(), map[string]any{
		"id":           float64(123),
		"name":         "Example",
		"description":  "This is an example tasklist.",
		"project_id":   float64(123),
		"milestone_id": float64(789),
	})
}

func TestTasklistDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTasklistGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTasklistList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

func TestTasklistListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistListByProject.String(), map[string]any{
		"search_term": "test",
		"project_id":  float64(123),
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
