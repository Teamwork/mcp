package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestLinkCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkCreate.String(), map[string]any{
		"code":                "http://example.com",
		"project_id":          float64(123),
		"title":               "Example",
		"description":         "Example message body",
		"tag_ids":             []float64{1, 2, 3},
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestLinkUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkUpdate.String(), map[string]any{
		"id":                  float64(123),
		"code":                "http://example.com",
		"title":               "Example",
		"description":         "Example message body",
		"tag_ids":             []float64{1, 2, 3},
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestLinkDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestLinkGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestLinkList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkList.String(), map[string]any{
		"search_term":    "test",
		"project_id":     float64(123),
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
