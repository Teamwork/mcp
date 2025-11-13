package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCommentCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreate.String(), map[string]any{
		"object": map[string]any{
			"type": "tasks",
			"id":   float64(123),
		},
		"body":         "Example",
		"content_type": "TEXT",
	})
}

func TestCommentUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentUpdate.String(), map[string]any{
		"id":           float64(123),
		"body":         "Example",
		"content_type": "TEXT",
	})
}

func TestCommentDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByFileVersion(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentListByFileVersion.String(), map[string]any{
		"search_term":     "test",
		"file_version_id": float64(123),
		"updated_after":   "2025-01-01T00:00:00Z",
		"page":            float64(1),
		"page_size":       float64(10),
	})
}

func TestCommentListByMilestone(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentListByMilestone.String(), map[string]any{
		"search_term":   "test",
		"milestone_id":  float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByNotebook(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentListByNotebook.String(), map[string]any{
		"search_term":   "test",
		"notebook_id":   float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByTask(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentListByTask.String(), map[string]any{
		"search_term":   "test",
		"task_id":       float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}
