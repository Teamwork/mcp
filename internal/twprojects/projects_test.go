package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCreate.String(), map[string]any{
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	})
}

func TestProjectUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	})
}

func TestProjectDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
