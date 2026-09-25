package twdesk_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

// TestSourceListRequest pins the route and checks the body is handed back as
// the endpoint answered it: the endpoint keys the rows ticketsources, which
// the SDK's typed response spells differently.
func TestSourceListRequest(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"ticketsources":[{"id":7,"name":"Email","isCustom":false}]}`))
	defer cleanup()

	text := lookupResultText(t, mcpServer, twdesk.MethodSourceList, map[string]any{
		"page": nil, "pageSize": float64(100), "orderBy": nil, "orderDirection": nil, "fields": nil,
	})

	requestURL := lastRequestURL()
	if got, want := requestURL.Path, "/desk/api/v2/ticketsources.json"; got != want {
		t.Errorf("unexpected request path: got %q, want %q", got, want)
	}
	if got := requestURL.Query().Get("pageSize"); got != "100" {
		t.Errorf("query parameter \"pageSize\": got %q, want %q", got, "100")
	}
	if !strings.Contains(text, `"ticketsources"`) || !strings.Contains(text, `"Email"`) {
		t.Errorf("result should carry the endpoint's rows untouched, got %s", text)
	}
}
