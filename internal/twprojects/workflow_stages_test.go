package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestWorkflowStageCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"stage":{"id":456,"name":"In Progress"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageCreate.String(), map[string]any{
		"workflow_id": float64(123),
		"name":        "In Progress",
	})
}

func TestWorkflowStageUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageUpdate.String(), map[string]any{
		"workflow_id": float64(123),
		"id":          float64(456),
		"name":        "Updated Stage",
	})
}

func TestWorkflowStageDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageDelete.String(), map[string]any{
		"workflow_id":           float64(123),
		"id":                    float64(456),
		"map_tasks_to_stage_id": float64(789),
	})
}

func TestWorkflowStageTaskMove(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageTaskMove.String(), map[string]any{
		"workflow_id": float64(123),
		"stage_id":    float64(456),
		"task_id":     float64(789),
	})
}

func TestWorkflowStageGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"stage":{"id":456,"name":"In Progress"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageGet.String(), map[string]any{
		"workflow_id": float64(123),
		"id":          float64(456),
	})
}

func TestWorkflowStageList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"stages":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageList.String(), map[string]any{
		"workflow_id": float64(123),
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
