package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// calendarsBody is the calendars endpoint's answer, which the events tool reads
// when the caller names no calendar.
func calendarsBody(calendars string) []byte {
	return []byte(`{"calendars":[` + calendars + `]}`)
}

const (
	blockedTimeCalendar = `{"id":11,"name":"blocked_time","type":"blocked_time"}`
	googleCalendar      = `{"id":22,"name":"john@example.com","type":"google"}`
	holidayCalendar     = `{"id":33,"name":"Holidays","type":"holiday"}`
	teamCalendar        = `{"id":44,"name":"Team","type":"event"}`
)

// TestCalendarEventListDefaultsToTheUsersCalendar pins the calendar_id lookup:
// omitting it must resolve to the connected integration, falling back to time
// blocking, instead of failing. The recorded requests are the only evidence,
// since the mock answers the events call with the same body either way.
func TestCalendarEventListDefaultsToTheUsersCalendar(t *testing.T) {
	tests := []struct {
		name      string
		calendars string
		wantPath  string
	}{{
		name:      "connected integration wins",
		calendars: blockedTimeCalendar + `,` + googleCalendar,
		wantPath:  "/projects/api/v3/calendars/22/events.json",
	}, {
		name:      "time blocking is the fallback",
		calendars: holidayCalendar + `,` + blockedTimeCalendar,
		wantPath:  "/projects/api/v3/calendars/11/events.json",
	}, {
		name:      "a lone calendar answers",
		calendars: teamCalendar,
		wantPath:  "/projects/api/v3/calendars/44/events.json",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{{
				Match:  "/calendars.json",
				Status: http.StatusOK,
				Body:   calendarsBody(tt.calendars),
			}}, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{})

			if len(*recorded) != 2 {
				t.Fatalf("expected a calendars lookup then an events request, got %d requests", len(*recorded))
			}
			if got := (*recorded)[0].URL.Query().Get("fields[calendars]"); got != "id,name,type" {
				t.Errorf("expected the lookup to select id,name,type but got %q", got)
			}
			if got := (*recorded)[1].URL.Path; got != tt.wantPath {
				t.Errorf("expected the events request to hit %s but got %s", tt.wantPath, got)
			}
		})
	}
}

// TestCalendarEventListNamedCalendarSkipsTheLookup pins that a caller naming a
// calendar still costs one request.
func TestCalendarEventListNamedCalendarSkipsTheLookup(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Match:  "/calendars.json",
		Status: http.StatusOK,
		Body:   calendarsBody(googleCalendar),
	}}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{
		"calendar_id": float64(123),
	})

	if len(*recorded) != 1 {
		t.Fatalf("expected a single events request, got %d", len(*recorded))
	}
	if got := (*recorded)[0].URL.Path; got != "/projects/api/v3/calendars/123/events.json" {
		t.Errorf("unexpected events path %s", got)
	}
}

// TestCalendarEventListUnresolvedCalendarReportsCandidates pins that an account
// the rule cannot decide for gets the candidate list rather than a guess, so the
// model can name one without a second tool.
func TestCalendarEventListUnresolvedCalendarReportsCandidates(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Match:  "/calendars.json",
		Status: http.StatusOK,
		Body:   calendarsBody(holidayCalendar + `,` + teamCalendar),
	}}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if !toolResult.IsError {
				t.Fatal("expected an error tool result when no calendar can be picked")
			}
			text, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("unexpected content type: %T", toolResult.Content[0])
			}
			if !strings.Contains(text.Text, `"id":44`) {
				t.Errorf("expected the candidates in the message but got %q", text.Text)
			}
		}))

	if len(*recorded) != 1 {
		t.Errorf("expected no events request, got %d requests", len(*recorded))
	}
}

// TestCalendarEventListLookupFailureIsAToolResult pins that a failed lookup is
// reported like any other API failure instead of becoming a Go error, which
// would reach the model as a transport failure with no status to read.
func TestCalendarEventListLookupFailureIsAToolResult(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusForbidden, []byte(`{"errors":[{"title":"nope"}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if !toolResult.IsError {
				t.Error("expected the failed calendars lookup to be an error tool result")
			}
		}))
}
