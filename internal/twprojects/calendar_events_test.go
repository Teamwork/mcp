package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCalendarEventList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{
		"calendar_id":        float64(123),
		"started_after_date": "2023-01-01",
		"ended_before_date":  "2023-12-31",
		"limit":              float64(25),
		"cursor":             "abc123",
	})
}

// TestCalendarEventListVerboseRestrictsSideloadFields pins the sideload field
// restrictions on the verbose path; without them every attendee returns a full
// user record.
func TestCalendarEventListVerboseRestrictsSideloadFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{
		"calendar_id": float64(123),
	})

	query := lastURL.Query()
	if len(query["include"]) == 0 {
		t.Fatalf("expected verbose mode to sideload (raw query: %s)", lastURL.RawQuery)
	}
	for param, want := range map[string]string{
		"fields[users]":    "id,firstName,lastName",
		"fields[projects]": "id,name",
		"fields[tasks]":    "id,name",
		"fields[timelogs]": "id,description,minutes",
	} {
		if got := query.Get(param); got != want {
			t.Errorf("expected %s=%q in request query but got %q", param, want, got)
		}
	}
	// verbose keeps the full event body.
	if got := query.Get("fields[calendarsEvents]"); got != "" {
		t.Errorf("expected no event field restriction in verbose mode but got %q", got)
	}
}
