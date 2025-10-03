package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestUserCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreate.String(), map[string]any{
		"name":       "Example",
		"first_name": "First",
		"last_name":  "Last",
		"title":      "Mr.",
		"email":      "example@test.com",
		"admin":      true,
		"type":       "account",
		"company_id": float64(456),
	})
}

func TestUserUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserUpdate.String(), map[string]any{
		"id":         float64(123),
		"name":       "Example",
		"first_name": "First",
		"last_name":  "Last",
		"title":      "Mr.",
		"email":      "example@test.com",
		"admin":      true,
		"type":       "account",
		"company_id": float64(456),
	})
}

func TestUserDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestUserGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestUserGetMe(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserGetMe.String(), map[string]any{})
}

func TestUserList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserList.String(), map[string]any{
		"search_term": "test",
		"type":        "account",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

func TestUserListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserListByProject.String(), map[string]any{
		"project_id":  float64(123),
		"search_term": "test",
		"type":        "account",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
