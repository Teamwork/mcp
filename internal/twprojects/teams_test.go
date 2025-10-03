package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTeamCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamCreate.String(), map[string]any{
		"name":           "Example",
		"handle":         "example",
		"description":    "Example description",
		"parent_team_id": float64(123),
		"company_id":     float64(456),
		"project_id":     float64(789),
		"user_ids": []any{
			float64(1),
			float64(2),
			float64(3),
		},
	})
}

func TestTeamUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamUpdate.String(), map[string]any{
		"id":             float64(123),
		"name":           "Example",
		"handle":         "example",
		"description":    "Example description",
		"parent_team_id": float64(123),
		"company_id":     float64(456),
		"project_id":     float64(789),
		"user_ids": []any{
			float64(1),
			float64(2),
			float64(3),
		},
	})
}

func TestTeamDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTeamGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTeamList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

func TestTeamListByCompany(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamListByCompany.String(), map[string]any{
		"company_id":  float64(123),
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

func TestTeamListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamListByProject.String(), map[string]any{
		"project_id":  float64(123),
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
