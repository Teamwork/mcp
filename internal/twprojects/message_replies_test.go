package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestMessageReplyCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"postId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyCreate.String(), map[string]any{
		"message_id":          float64(123),
		"body":                "Example message reply body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestMessageReplyUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyUpdate.String(), map[string]any{
		"id":                  float64(123),
		"body":                "Example message reply body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestMessageReplyDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageReplyGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageReplyList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyList.String(), map[string]any{
		"search_term": "test",
		"message_ids": []float64{1, 2, 3},
		"project_ids": []float64{1, 2, 3},
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
