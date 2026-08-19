package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestActivityList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodActivityList.String(), map[string]any{
		"start_date": "2023-10-01T00:00:00Z",
		"end_date":   "2023-10-31T23:59:59Z",
		"log_item_types": []any{
			"message",
			"comment",
			"task",
			"tasklist",
			"taskgroup",
			"milestone",
			"file",
			"form",
			"notebook",
			"timelog",
			"task_comment",
			"notebook_comment",
			"file_comment",
			"link_comment",
			"milestone_comment",
			"project",
			"link",
			"billingInvoice",
			"risk",
			"projectUpdate",
			"reacted",
			"budget",
		},
		"page":      float64(1),
		"page_size": float64(10),
	})
}

func TestActivityListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodActivityList.String(), map[string]any{
		"project_id": float64(123),
		"start_date": "2023-10-01T00:00:00Z",
		"end_date":   "2023-10-31T23:59:59Z",
		"log_item_types": []any{
			"message",
			"comment",
			"task",
			"tasklist",
			"taskgroup",
			"milestone",
			"file",
			"form",
			"notebook",
			"timelog",
			"task_comment",
			"notebook_comment",
			"file_comment",
			"link_comment",
			"milestone_comment",
			"project",
			"link",
			"billingInvoice",
			"risk",
			"projectUpdate",
			"reacted",
			"budget",
		},
		"page":      float64(1),
		"page_size": float64(10),
	})
}

// TestActivityListItemIDsReachTheWire pins the item_ids filter on the query
// string: the mock replies with the same canned body whether or not the filter
// is forwarded, so a dropped argument is otherwise invisible.
func TestActivityListItemIDsReachTheWire(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodActivityList.String(), map[string]any{
		"item_ids":       []any{float64(777), float64(12345)},
		"log_item_types": []any{"task"},
	})

	if got, want := lastURL.Query().Get("itemIds"), "777,12345"; got != want {
		t.Errorf("expected itemIds=%q in request query but got %q (raw query: %s)", want, got, lastURL.RawQuery)
	}
}
