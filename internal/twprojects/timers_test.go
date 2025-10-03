package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTimerCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timer":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerCreate.String(), map[string]any{
		"description":         "Example timer description",
		"billable":            true,
		"running":             true,
		"seconds":             float64(3600), // 1 hour
		"stop_running_timers": true,
		"project_id":          float64(123),
		"task_id":             float64(456),
	})
}

func TestTimerUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerUpdate.String(), map[string]any{
		"id":          float64(123),
		"description": "Example timer description",
		"billable":    true,
		"running":     true,
		"project_id":  float64(123),
		"task_id":     float64(456),
	})
}

func TestTimerPause(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerPause.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerResume(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerResume.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerComplete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerComplete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerList.String(), map[string]any{
		"user_id":             float64(123),
		"task_id":             float64(456),
		"project_id":          float64(789),
		"running_timers_only": true,
		"page":                float64(1),
		"page_size":           float64(10),
	})
}
