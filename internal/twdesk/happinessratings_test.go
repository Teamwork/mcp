package twdesk_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

// TestHappinessRatingOptionListRequest pins the route and checks the options
// come back with their names and scores, which is what a caller needs to pick
// the installation-specific IDs twdesk-search_tickets filters on.
func TestHappinessRatingOptionListRequest(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"happinessratingoptions":[{"id":31,"name":"Great","score":3,"color":"#00aa00"}]}`))
	defer cleanup()

	text := lookupResultText(t, mcpServer, twdesk.MethodHappinessRatingOptionList, map[string]any{
		"page": nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil, "fields": nil,
	})

	requestURL := lastRequestURL()
	if got, want := requestURL.Path, "/desk/api/v2/happinessratingoptions.json"; got != want {
		t.Errorf("unexpected request path: got %q, want %q", got, want)
	}
	for _, want := range []string{`"happinessratingoptions"`, `"id":31`, `"score":3`} {
		if !strings.Contains(text, want) {
			t.Errorf("result should contain %s, got %s", want, text)
		}
	}
}
