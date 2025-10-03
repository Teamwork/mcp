package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTaskCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"task":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), map[string]any{
		"name":              "Example",
		"tasklist_id":       float64(123),
		"description":       "This is an example task.",
		"priority":          "high",
		"progress":          float64(50),
		"start_date":        "2023-10-01",
		"due_date":          "2023-10-15",
		"estimated_minutes": float64(120),
		"parent_task_id":    float64(456),
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"team_ids":    []float64{4, 5},
			"company_ids": []float64{6, 7},
		},
		"tag_ids": []float64{1, 2, 3},
		"predecessors": []map[string]any{
			{
				"task_id": float64(456),
				"type":    "start",
			},
			{
				"task_id": float64(789),
				"type":    "complete",
			},
		},
	})
}

func TestTaskUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":                float64(123),
		"name":              "Example",
		"tasklist_id":       float64(123),
		"description":       "This is an example task.",
		"priority":          "high",
		"progress":          float64(50),
		"start_date":        "2023-10-01",
		"due_date":          "2023-10-15",
		"estimated_minutes": float64(120),
		"parent_task_id":    float64(456),
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"team_ids":    []float64{4, 5},
			"company_ids": []float64{6, 7},
		},
		"tag_ids": []float64{1, 2, 3},
		"predecessors": []map[string]any{
			{
				"task_id": float64(456),
				"type":    "start",
			},
			{
				"task_id": float64(789),
				"type":    "complete",
			},
		},
	})
}

func TestTaskDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTaskGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTaskList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term":       "test",
		"tag_ids":           []float64{1, 2, 3},
		"match_all_tags":    true,
		"page":              float64(1),
		"page_size":         float64(10),
		"assignee_user_ids": []float64{4, 5, 6},
	})
}

func TestTaskListByTasklist(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskListByTasklist.String(), map[string]any{
		"tasklist_id":       float64(123),
		"search_term":       "test",
		"tag_ids":           []float64{1, 2, 3},
		"match_all_tags":    true,
		"page":              float64(1),
		"page_size":         float64(10),
		"assignee_user_ids": []float64{4, 5, 6},
	})
}

func TestTaskListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskListByProject.String(), map[string]any{
		"project_id":        float64(123),
		"search_term":       "test",
		"tag_ids":           []float64{1, 2, 3},
		"match_all_tags":    true,
		"page":              float64(1),
		"page_size":         float64(10),
		"assignee_user_ids": []float64{4, 5, 6},
	})
}
