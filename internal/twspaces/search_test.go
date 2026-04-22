//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestSearch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"totalResults":1,"results":[{"pageId":10,"title":"Getting Started","slug":"getting-started","matched":{"content":["Welcome to our <em>docs</em>"]},"space":{"id":1,"title":"Engineering"},"tags":[]}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSearch.String(), map[string]any{
		"query": "docs",
	})
}

func TestSearchWithFilters(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"totalResults":0,"results":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodSearch.String(), map[string]any{
		"query":    "api reference",
		"spaceIds": []any{float64(1), float64(2)},
		"limit":    float64(10),
		"offset":   float64(0),
	})
}
