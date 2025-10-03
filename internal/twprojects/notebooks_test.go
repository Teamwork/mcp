package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestNotebookCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"notebook":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodNotebookCreate.String(), map[string]any{
		"name":        "Example",
		"project_id":  float64(123),
		"description": "Example notebook description",
		"contents":    "This is the content of the notebook.",
		"type":        "MARKDOWN",
		"tag_ids":     []float64{10, 11, 12},
	})
}

func TestNotebookUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodNotebookUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "Example notebook description",
		"contents":    "This is the content of the notebook.",
		"type":        "MARKDOWN",
		"tag_ids":     []float64{10, 11, 12},
	})
}

func TestNotebookDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodNotebookDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestNotebookGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodNotebookGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestNotebookList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodNotebookList.String(), map[string]any{
		"project_ids":      []float64{123, 456},
		"search_term":      "test",
		"tag_ids":          []float64{1, 2, 3},
		"match_all_tags":   true,
		"include_contents": true,
		"page":             float64(1),
		"page_size":        float64(10),
	})
}
