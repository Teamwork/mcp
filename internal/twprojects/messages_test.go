package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestMessageCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"messageId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageCreate.String(), map[string]any{
		"title":               "Example",
		"project_id":          float64(123),
		"body":                "Example message body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestMessageUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageUpdate.String(), map[string]any{
		"id":                  float64(123),
		"title":               "Example",
		"body":                "Example message body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestMessageDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageList.String(), map[string]any{
		"search_term":    "test",
		"project_ids":    []float64{1, 2, 3},
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
