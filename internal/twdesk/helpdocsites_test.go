//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestHelpDocSiteGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocssite":{"id":7,"name":"Support Centre","subdomain":"support"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocSiteGet.String(), map[string]any{
		"id":     float64(7),
		"fields": nil,
	})
}

func TestHelpDocSiteList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocssites":[{"id":7,"name":"Support Centre","subdomain":"support"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocSiteList.String(), map[string]any{
		"name":           []any{"Support Centre"},
		"subdomain":      []any{"support"},
		"page":           float64(1),
		"pageSize":       float64(10),
		"orderBy":        "name",
		"orderDirection": "asc",
		"fields":         []any{"id", "name", "subdomain"},
	})
}

func TestHelpDocSiteListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"helpdocssites":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodHelpDocSiteList.String(), map[string]any{
		"name":           nil,
		"subdomain":      nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}
