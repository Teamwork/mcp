package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestSearch(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term":             "test",
		"project_id":              float64(123),
		"include_completed_items": true,
		"updated_after":           "2023-01-01T00:00:00Z",
		"extended_search":         true,
		"cursor":                  "c858b04ba8b066bcb4f83727c23de6e9238de642",
		"limit":                   float64(10),
	})
}
