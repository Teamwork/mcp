package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectCategoryCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"categoryId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCategoryCreate.String(), map[string]any{
		"name":      "Example",
		"color":     "#FF5733",
		"parent_id": float64(456),
	})
}

func TestProjectCategoryUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCategoryUpdate.String(), map[string]any{
		"id":        float64(123),
		"name":      "Example",
		"color":     "#FF5733",
		"parent_id": float64(456),
	})
}

func TestProjectCategoryDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCategoryDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectCategoryGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCategoryGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectCategoryList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCategoryList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
