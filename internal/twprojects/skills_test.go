package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestSkillCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"skill":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSkillCreate.String(), map[string]any{
		"name":     "Example",
		"user_ids": []float64{1, 2, 3},
	})
}

func TestSkillUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSkillUpdate.String(), map[string]any{
		"id":       float64(123),
		"name":     "Example",
		"user_ids": []float64{1, 2, 3},
	})
}

func TestSkillDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSkillDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestSkillGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSkillGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestSkillList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSkillList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
