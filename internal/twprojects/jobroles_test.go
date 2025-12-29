package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestJobRoleCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"jobRole":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleCreate.String(), map[string]any{
		"name": "Example",
	})
}

func TestJobRoleUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleUpdate.String(), map[string]any{
		"id":   float64(123),
		"name": "Example",
	})
}

func TestJobRoleDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestJobRoleGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestJobRoleList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
