package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectTemplateCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectTemplateCreate.String(), map[string]any{
		"name":        "Example",
		"description": "This is an example project template.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	})
}

func TestProjectTemplateList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectTemplateList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
