package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestWorkflowCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"workflow":{"id":123,"name":"Example"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowCreate.String(), map[string]any{
		"name": "Example",
	})
}

func TestWorkflowUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowUpdate.String(), map[string]any{
		"id":   float64(123),
		"name": "Updated Example",
	})
}

func TestWorkflowDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestWorkflowProjectLink(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowProjectLink.String(), map[string]any{
		"project_id":  float64(123),
		"workflow_id": float64(456),
	})
}

func TestWorkflowGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"workflow":{"id":123,"name":"Example"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestWorkflowList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"workflows":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
