package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestMilestoneCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"milestoneId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneCreate.String(), map[string]any{
		"name":        "Example",
		"project_id":  float64(123),
		"description": "Example milestone description",
		"due_date":    "20231231",
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
		"tasklist_ids": []float64{8, 9},
		"tag_ids":      []float64{10, 11, 12},
	})
}

func TestMilestoneUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "Example milestone description",
		"due_date":    "20231231",
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
		"tasklist_ids": []float64{8, 9},
		"tag_ids":      []float64{10, 11, 12},
	})
}

func TestMilestoneDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMilestoneGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMilestoneList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}

func TestMilestoneListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneListByProject.String(), map[string]any{
		"project_id":     float64(123),
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
