package twprojects_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// tlCols mirrors the ten published time-aggregate columns for decoding the tool
// response in tests.
type tlCols struct {
	LoggedMinutes           int64   `json:"loggedMinutes"`
	LoggedHours             float64 `json:"loggedHours"`
	BillableMinutes         int64   `json:"billableMinutes"`
	BillableHours           float64 `json:"billableHours"`
	NonBillableMinutes      int64   `json:"nonBillableMinutes"`
	NonBillableHours        float64 `json:"nonBillableHours"`
	BilledMinutes           int64   `json:"billedMinutes"`
	BilledHours             float64 `json:"billedHours"`
	UnbilledBillableMinutes int64   `json:"unbilledBillableMinutes"`
	UnbilledBillableHours   float64 `json:"unbilledBillableHours"`
}

type tlResult struct {
	Scope struct {
		GroupBy   string `json:"groupBy"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	} `json:"scope"`
	Totals struct {
		tlCols
		GroupCount int64 `json:"groupCount"`
	} `json:"totals"`
	Groups []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		tlCols
	} `json:"groups"`
	Periods []struct {
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		tlCols
	} `json:"periods"`
}

// decodeSummary extracts and decodes the summarize_timelogs JSON payload from a
// successful tool result.
func decodeSummary(t *testing.T, result mcp.Result) tlResult {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if toolResult.IsError {
		t.Fatalf("tool returned an error: %v", toolResult.Content)
	}
	if len(toolResult.Content) == 0 {
		t.Fatalf("tool result has no content")
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	var decoded tlResult
	if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
		t.Fatalf("failed to decode summary payload %q: %v", text.Text, err)
	}
	return decoded
}

// expectToolError runs the tool and asserts it returned an error result whose
// message contains want.
func expectToolError(t *testing.T, want string) testutil.ExecuteToolRequestOption {
	t.Helper()
	return testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Fatalf("expected an error result, got success: %v", toolResult.Content)
		}
		if want == "" {
			return
		}
		var msg string
		if len(toolResult.Content) > 0 {
			if text, ok := toolResult.Content[0].(*mcp.TextContent); ok {
				msg = text.Text
			}
		}
		if !strings.Contains(msg, want) {
			t.Errorf("expected error message to contain %q, got %q", want, msg)
		}
	})
}

// assertReconciles asserts that the row columns (groups or periods) sum to the
// totals block exactly, in minutes (minutes are the authoritative values).
func assertReconciles(t *testing.T, res tlResult) {
	t.Helper()

	var logged, billable, nonBillable, billed, unbilled int64
	rows := make([]tlCols, 0, len(res.Groups)+len(res.Periods))
	for _, g := range res.Groups {
		rows = append(rows, g.tlCols)
	}
	for _, p := range res.Periods {
		rows = append(rows, p.tlCols)
	}
	for _, g := range rows {
		logged += g.LoggedMinutes
		billable += g.BillableMinutes
		nonBillable += g.NonBillableMinutes
		billed += g.BilledMinutes
		unbilled += g.UnbilledBillableMinutes
	}
	if logged != res.Totals.LoggedMinutes {
		t.Errorf("logged minutes: Σgroups=%d totals=%d", logged, res.Totals.LoggedMinutes)
	}
	if billable != res.Totals.BillableMinutes {
		t.Errorf("billable minutes: Σgroups=%d totals=%d", billable, res.Totals.BillableMinutes)
	}
	if nonBillable != res.Totals.NonBillableMinutes {
		t.Errorf("non-billable minutes: Σgroups=%d totals=%d", nonBillable, res.Totals.NonBillableMinutes)
	}
	if billed != res.Totals.BilledMinutes {
		t.Errorf("billed minutes: Σgroups=%d totals=%d", billed, res.Totals.BilledMinutes)
	}
	if unbilled != res.Totals.UnbilledBillableMinutes {
		t.Errorf("unbilled-billable minutes: Σgroups=%d totals=%d", unbilled, res.Totals.UnbilledBillableMinutes)
	}
	if int64(len(rows)) != res.Totals.GroupCount {
		t.Errorf("groupCount=%d but %d rows returned", res.Totals.GroupCount, len(rows))
	}
}

func TestSummarizeTimelogsByUser(t *testing.T) {
	body := []byte(`{
		"meta": {"page": {"hasMore": false}},
		"time": {"users": [
			{"loggedTime": 810, "billableTime": 600, "nonBillableTime": 210, "billedTime": 120,
			 "estimatedTime": 0, "user": {"id": 525154, "type": "users"}},
			{"loggedTime": 750, "billableTime": 600, "nonBillableTime": 150, "billedTime": 120,
			 "estimatedTime": 0, "user": {"id": 999, "type": "users"}}
		]},
		"included": {"users": {
			"525154": {"id": 525154, "firstName": "Gary", "lastName": "Meehan"},
			"999": {"id": 999, "firstName": "Jane", "lastName": "Doe"}
		}}
	}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if res.Scope.GroupBy != "user" {
			t.Errorf("expected groupBy user, got %q", res.Scope.GroupBy)
		}
		if res.Scope.StartDate != "2026-07-01" || res.Scope.EndDate != "2026-07-31" {
			t.Errorf("scope window not echoed back: %+v", res.Scope)
		}
		if len(res.Groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(res.Groups))
		}
		// Rows preserve first-seen order.
		if res.Groups[0].ID != 525154 || res.Groups[0].Name != "Gary Meehan" {
			t.Errorf("group[0] name join wrong: %+v", res.Groups[0])
		}
		if res.Groups[1].ID != 999 || res.Groups[1].Name != "Jane Doe" {
			t.Errorf("group[1] name join wrong: %+v", res.Groups[1])
		}
		// Per-group columns and hour conversion.
		if res.Groups[0].LoggedMinutes != 810 || res.Groups[0].LoggedHours != 13.5 {
			t.Errorf("group[0] logged wrong: %d min / %v h", res.Groups[0].LoggedMinutes, res.Groups[0].LoggedHours)
		}
		// unbilledBillable = billable - billed = 600 - 120 = 480.
		if res.Groups[0].UnbilledBillableMinutes != 480 {
			t.Errorf("group[0] unbilledBillable wrong: %d", res.Groups[0].UnbilledBillableMinutes)
		}
		// Totals (matches PRD example: 26h/20h/6h/4h/16h).
		if res.Totals.LoggedMinutes != 1560 || res.Totals.LoggedHours != 26 {
			t.Errorf("totals logged wrong: %d min / %v h", res.Totals.LoggedMinutes, res.Totals.LoggedHours)
		}
		if res.Totals.BillableMinutes != 1200 || res.Totals.BilledMinutes != 240 || res.Totals.UnbilledBillableMinutes != 960 {
			t.Errorf("totals billable/billed/unbilled wrong: %+v", res.Totals)
		}
		assertReconciles(t, res)
	}))
}

func TestSummarizeTimelogsByProject(t *testing.T) {
	body := []byte(`{
		"meta": {"page": {"hasMore": false}},
		"time": {"projects": [
			{"loggedTime": 300, "billableTime": 300, "nonBillableTime": 0, "billedTime": 60,
			 "estimatedTime": 0, "project": {"id": 1, "type": "projects"}},
			{"loggedTime": 120, "billableTime": 0, "nonBillableTime": 120, "billedTime": 0,
			 "estimatedTime": 0, "project": {"id": 2, "type": "projects"}}
		]},
		"included": {"projects": {
			"1": {"id": 1, "name": "Website Revamp"},
			"2": {"id": 2, "name": "Internal"}
		}}
	}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date":  "2026-07-01",
		"end_date":    "2026-07-31",
		"group_by":    "project",
		"project_ids": []float64{1, 2},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if res.Scope.GroupBy != "project" {
			t.Errorf("expected groupBy project, got %q", res.Scope.GroupBy)
		}
		if len(res.Groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(res.Groups))
		}
		if res.Groups[0].ID != 1 || res.Groups[0].Name != "Website Revamp" {
			t.Errorf("group[0] wrong: %+v", res.Groups[0])
		}
		if res.Groups[1].ID != 2 || res.Groups[1].Name != "Internal" {
			t.Errorf("group[1] wrong: %+v", res.Groups[1])
		}
		if res.Totals.LoggedMinutes != 420 {
			t.Errorf("totals logged wrong: %d", res.Totals.LoggedMinutes)
		}
		assertReconciles(t, res)
	}))
}

func TestSummarizeTimelogsPaginatesMultiplePages(t *testing.T) {
	page1 := []byte(`{
		"meta": {"page": {"hasMore": true}},
		"time": {"users": [
			{"loggedTime": 600, "billableTime": 600, "nonBillableTime": 0, "billedTime": 0,
			 "estimatedTime": 0, "user": {"id": 1, "type": "users"}}
		]},
		"included": {"users": {"1": {"id": 1, "firstName": "Alice", "lastName": "One"}}}
	}`)
	page2 := []byte(`{
		"meta": {"page": {"hasMore": false}},
		"time": {"users": [
			{"loggedTime": 300, "billableTime": 0, "nonBillableTime": 300, "billedTime": 0,
			 "estimatedTime": 0, "user": {"id": 2, "type": "users"}}
		]},
		"included": {"users": {"2": {"id": 2, "firstName": "Bob", "lastName": "Two"}}}
	}`)

	mcpServer := testutil.ProjectsMCPServerSequencedMock(t, http.StatusOK, page1, page2)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Groups) != 2 {
			t.Fatalf("expected 2 groups aggregated across pages, got %d", len(res.Groups))
		}
		if res.Totals.LoggedMinutes != 900 {
			t.Errorf("expected 900 logged minutes across pages, got %d", res.Totals.LoggedMinutes)
		}
		if res.Totals.GroupCount != 2 {
			t.Errorf("expected groupCount 2, got %d", res.Totals.GroupCount)
		}
		assertReconciles(t, res)
	}))
}

func TestSummarizeTimelogsPageCapFailsLoudly(t *testing.T) {
	// A single body whose hasMore is always true makes the loop run forever; the
	// sequenced mock repeats it, so the 10-page cap must trip.
	alwaysMore := []byte(`{
		"meta": {"page": {"hasMore": true}},
		"time": {"users": [
			{"loggedTime": 10, "billableTime": 10, "nonBillableTime": 0, "billedTime": 0,
			 "estimatedTime": 0, "user": {"id": 1, "type": "users"}}
		]},
		"included": {"users": {"1": {"id": 1, "firstName": "Alice", "lastName": "One"}}}
	}`)

	mcpServer := testutil.ProjectsMCPServerSequencedMock(t, http.StatusOK, alwaysMore)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, expectToolError(t, "10-page limit"))
}

func TestSummarizeTimelogsEmptyWindowReturnsZeros(t *testing.T) {
	body := []byte(`{"meta": {"page": {"hasMore": false}}, "time": {"users": []}, "included": {"users": {}}}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Groups) != 0 {
			t.Errorf("expected no groups, got %d", len(res.Groups))
		}
		if res.Totals.GroupCount != 0 {
			t.Errorf("expected groupCount 0, got %d", res.Totals.GroupCount)
		}
		if res.Totals.LoggedMinutes != 0 || res.Totals.LoggedHours != 0 || res.Totals.BillableMinutes != 0 {
			t.Errorf("expected zero totals, got %+v", res.Totals)
		}
	}))
}

func TestSummarizeTimelogsMissingSideloadFallsBackToSyntheticName(t *testing.T) {
	body := []byte(`{
		"meta": {"page": {"hasMore": false}},
		"time": {"users": [
			{"loggedTime": 60, "billableTime": 60, "nonBillableTime": 0, "billedTime": 0,
			 "estimatedTime": 0, "user": {"id": 777, "type": "users"}}
		]},
		"included": {"users": {}}
	}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Groups) != 1 {
			t.Fatalf("expected the row to be kept despite the missing sideload, got %d groups", len(res.Groups))
		}
		if res.Groups[0].Name != "user 777" {
			t.Errorf("expected fallback name %q, got %q", "user 777", res.Groups[0].Name)
		}
	}))
}

func TestSummarizeTimelogsRejectsReversedWindow(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-31",
		"end_date":   "2026-07-01",
	}, expectToolError(t, "must be on or before"))
}

func TestSummarizeTimelogsPlanGate403(t *testing.T) {
	body := []byte(`{"errors": [{"detail": "You do not have permission to view this report."}]}`)

	mcpServer := mcpServerMock(t, http.StatusForbidden, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
	}, expectToolError(t, ""))
}

func TestSummarizeTimelogsByTask(t *testing.T) {
	body := []byte(`{
		"meta": {"page": {"hasMore": false}},
		"time": {"tasks": [
			{"loggedTime": 240, "billableTime": 240, "nonBillableTime": 0, "billedTime": 60,
			 "estimatedTime": 0, "task": {"id": 777, "type": "tasks"}},
			{"loggedTime": 90, "billableTime": 0, "nonBillableTime": 90, "billedTime": 0,
			 "estimatedTime": 0, "task": {"id": 778, "type": "tasks"}}
		]},
		"included": {"tasks": {
			"777": {"id": 777, "name": "Write the release notes"},
			"778": {"id": 778, "name": "Review the release notes"}
		}}
	}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
		"group_by":   "task",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if res.Scope.GroupBy != "task" {
			t.Errorf("expected groupBy task, got %q", res.Scope.GroupBy)
		}
		if len(res.Groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(res.Groups))
		}
		if res.Groups[0].ID != 777 || res.Groups[0].Name != "Write the release notes" {
			t.Errorf("group[0] name join wrong: %+v", res.Groups[0])
		}
		if res.Groups[1].ID != 778 || res.Groups[1].Name != "Review the release notes" {
			t.Errorf("group[1] name join wrong: %+v", res.Groups[1])
		}
		if res.Groups[0].LoggedMinutes != 240 || res.Groups[0].LoggedHours != 4 {
			t.Errorf("group[0] logged wrong: %d min / %v h", res.Groups[0].LoggedMinutes, res.Groups[0].LoggedHours)
		}
		// unbilledBillable = billable - billed = 240 - 60 = 180.
		if res.Groups[0].UnbilledBillableMinutes != 180 {
			t.Errorf("group[0] unbilledBillable wrong: %d", res.Groups[0].UnbilledBillableMinutes)
		}
		if res.Totals.LoggedMinutes != 330 || res.Totals.LoggedHours != 5.5 {
			t.Errorf("totals logged wrong: %d min / %v h", res.Totals.LoggedMinutes, res.Totals.LoggedHours)
		}
		assertReconciles(t, res)
	}))
}

// TestSummarizeTimelogsGroupByReachesTheWire pins the request each group_by
// value builds. The mocks reply with the same body whatever the query string, so
// a dimension, report variant or sideload that never reaches the API looks
// identical to a working one.
func TestSummarizeTimelogsGroupByReachesTheWire(t *testing.T) {
	tests := []struct {
		groupBy    string
		path       string
		reportType string
		include    string
		fieldsKey  string
		fields     string
	}{{
		groupBy:    "user",
		path:       "/projects/api/v3/time/report/user.json",
		reportType: "userloggedtime",
		include:    "users",
		fieldsKey:  "fields[users]",
		fields:     "id,firstName,lastName",
	}, {
		groupBy:    "project",
		path:       "/projects/api/v3/time/report/project.json",
		reportType: "projecttime",
		include:    "projects",
		fieldsKey:  "fields[projects]",
		fields:     "id,name",
	}, {
		groupBy:    "task",
		path:       "/projects/api/v3/time/report/task.json",
		reportType: "loggedtime",
		include:    "tasks",
		fieldsKey:  "fields[tasks]",
		fields:     "id,name",
	}}

	for _, tt := range tests {
		t.Run(tt.groupBy, func(t *testing.T) {
			body := []byte(`{"meta": {"page": {"hasMore": false}}, "time": {}, "included": {}}`)
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, body)

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
				"start_date": "2026-07-01",
				"end_date":   "2026-07-31",
				"group_by":   tt.groupBy,
			})

			if lastURL.Path != tt.path {
				t.Errorf("expected path %q, got %q", tt.path, lastURL.Path)
			}
			query := lastURL.Query()
			if got := query.Get("type"); got != tt.groupBy {
				t.Errorf("expected type=%s, got %q", tt.groupBy, got)
			}
			if got := query.Get("reportType"); got != tt.reportType {
				t.Errorf("expected reportType=%s, got %q", tt.reportType, got)
			}
			if got := query.Get("include"); got != tt.include {
				t.Errorf("expected include=%s, got %q", tt.include, got)
			}
			if got := query.Get(tt.fieldsKey); got != tt.fields {
				t.Errorf("expected %s=%s, got %q", tt.fieldsKey, tt.fields, got)
			}
		})
	}
}

func TestSummarizeTimelogsByWeek(t *testing.T) {
	// Opens mid-week: first period clipped to two days, second all zeros.
	body := []byte(`{
		"loggedTime": 1560, "billableTime": 1200, "nonBillableTime": 360, "billedTime": 240, "estimatedTime": 0,
		"dates": [
			{"startDate": "2026-08-01", "endDate": "2026-08-02", "loggedTime": 810, "billableTime": 600,
			 "nonBillableTime": 210, "billedTime": 120, "estimatedTime": 0},
			{"startDate": "2026-08-03", "endDate": "2026-08-09", "loggedTime": 0, "billableTime": 0,
			 "nonBillableTime": 0, "billedTime": 0, "estimatedTime": 0},
			{"startDate": "2026-08-10", "endDate": "2026-08-16", "loggedTime": 750, "billableTime": 600,
			 "nonBillableTime": 150, "billedTime": 120, "estimatedTime": 0}
		]
	}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-08-01",
		"end_date":   "2026-08-16",
		"group_by":   "week",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if res.Scope.GroupBy != "week" {
			t.Errorf("expected groupBy week, got %q", res.Scope.GroupBy)
		}
		if len(res.Groups) != 0 {
			t.Errorf("expected no groups for a period grouping, got %d", len(res.Groups))
		}
		if len(res.Periods) != 3 {
			t.Fatalf("expected 3 periods, got %d", len(res.Periods))
		}
		// Chronological order and the clipped window are kept.
		if res.Periods[0].StartDate != "2026-08-01" || res.Periods[0].EndDate != "2026-08-02" {
			t.Errorf("period[0] window wrong: %+v", res.Periods[0])
		}
		if res.Periods[0].LoggedMinutes != 810 || res.Periods[0].LoggedHours != 13.5 {
			t.Errorf("period[0] logged wrong: %d min / %v h", res.Periods[0].LoggedMinutes, res.Periods[0].LoggedHours)
		}
		// unbilledBillable = billable - billed = 600 - 120 = 480.
		if res.Periods[0].UnbilledBillableMinutes != 480 {
			t.Errorf("period[0] unbilledBillable wrong: %d", res.Periods[0].UnbilledBillableMinutes)
		}
		if res.Periods[1].LoggedMinutes != 0 || res.Periods[1].StartDate != "2026-08-03" {
			t.Errorf("empty period must be kept as zeros: %+v", res.Periods[1])
		}
		if res.Totals.LoggedMinutes != 1560 || res.Totals.LoggedHours != 26 {
			t.Errorf("totals logged wrong: %d min / %v h", res.Totals.LoggedMinutes, res.Totals.LoggedHours)
		}
		if res.Totals.GroupCount != 3 {
			t.Errorf("expected groupCount 3, got %d", res.Totals.GroupCount)
		}
		assertReconciles(t, res)
	}))
}

func TestSummarizeTimelogsByPeriodEmptyWindowReturnsZeros(t *testing.T) {
	body := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": []}`)

	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
		"group_by":   "month",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Periods) != 0 || len(res.Groups) != 0 {
			t.Errorf("expected no rows, got %d periods and %d groups", len(res.Periods), len(res.Groups))
		}
		if res.Totals.GroupCount != 0 || res.Totals.LoggedMinutes != 0 {
			t.Errorf("expected zero totals, got %+v", res.Totals)
		}
	}))
}

func TestSummarizeTimelogsByPeriodRejectsOrdering(t *testing.T) {
	for _, args := range []map[string]any{
		{"order_by": "loggedtime"},
		{"order_mode": "desc"},
	} {
		mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"dates": []}`))
		request := map[string]any{
			"start_date": "2026-07-01",
			"end_date":   "2026-07-31",
			"group_by":   "day",
		}
		for key, value := range args {
			request[key] = value
		}
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), request,
			expectToolError(t, "not accepted when group_by is day"))
	}
}

func TestSummarizeTimelogsByPeriodPlanGate403(t *testing.T) {
	body := []byte(`{"errors": [{"detail": "You do not have permission to view this report."}]}`)

	mcpServer := mcpServerMock(t, http.StatusForbidden, body)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2026-07-01",
		"end_date":   "2026-07-31",
		"group_by":   "week",
	}, expectToolError(t, ""))
}

// TestSummarizeTimelogsPeriodGroupByReachesTheWire pins the totals request each
// period value builds; the mocks reply the same body either way.
func TestSummarizeTimelogsPeriodGroupByReachesTheWire(t *testing.T) {
	for _, groupBy := range []string{"day", "week", "month"} {
		t.Run(groupBy, func(t *testing.T) {
			body := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
				"estimatedTime": 0, "dates": []}`)
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK, body)

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
				"start_date":                "2026-01-05",
				"end_date":                  "2026-08-16",
				"group_by":                  groupBy,
				"project_ids":               []float64{1, 2},
				"user_ids":                  []float64{3},
				"task_ids":                  []float64{4},
				"tasklist_ids":              []float64{5},
				"company_ids":               []float64{6},
				"team_ids":                  []float64{7},
				"timelog_tag_ids":           []float64{8, 9},
				"include_archived_projects": true,
			})

			if len(*urls) != 1 {
				t.Fatalf("expected a single request to the totals endpoint, got %d", len(*urls))
			}
			lastURL := (*urls)[0]
			if lastURL.Path != "/projects/api/v3/time/report/totals.json" {
				t.Errorf("expected the totals path, got %q", lastURL.Path)
			}
			query := lastURL.Query()
			want := map[string]string{
				"groupBy":                 groupBy,
				"startDate":               "2026-01-05",
				"endDate":                 "2026-08-16",
				"projectIds":              "1,2",
				"userIds":                 "3",
				"taskIds":                 "4",
				"tasklistIds":             "5",
				"companyIds":              "6",
				"teamIds":                 "7",
				"timelogTagIds":           "8,9",
				"includeArchivedProjects": "true",
			}
			for key, value := range want {
				if got := query.Get(key); got != value {
					t.Errorf("expected %s=%s, got %q", key, value, got)
				}
			}
			for _, key := range []string{"type", "reportType", "include", "page", "pageSize", "orderBy", "orderMode"} {
				if _, ok := query[key]; ok {
					t.Errorf("expected %s to be absent from the totals request, got %q", key, query.Get(key))
				}
			}
		})
	}
}

// TestSummarizeTimelogsPeriodSplitsTheWindowPerCalendarYear pins one request
// per calendar year, clipped to the caller's window. The endpoint buckets by
// day of year or month number, so one multi-year request would fold years.
func TestSummarizeTimelogsPeriodSplitsTheWindowPerCalendarYear(t *testing.T) {
	body := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": []}`)
	mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK, body)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2024-11-15",
		"end_date":   "2026-02-10",
		"group_by":   "month",
	})

	want := [][2]string{
		{"2024-11-15", "2024-12-31"},
		{"2025-01-01", "2025-12-31"},
		{"2026-01-01", "2026-02-10"},
	}
	if len(*urls) != len(want) {
		t.Fatalf("expected %d requests, one per calendar year, got %d", len(want), len(*urls))
	}
	for i, window := range want {
		query := (*urls)[i].Query()
		if query.Get("startDate") != window[0] || query.Get("endDate") != window[1] {
			t.Errorf("request %d: expected window %s to %s, got %s to %s",
				i, window[0], window[1], query.Get("startDate"), query.Get("endDate"))
		}
		if query.Get("groupBy") != "month" {
			t.Errorf("request %d: expected groupBy=month, got %q", i, query.Get("groupBy"))
		}
	}
}

// TestSummarizeTimelogsWeekSplitByNewYearIsMerged pins that the two halves of
// a week cut by the 1 January split come back as one row.
func TestSummarizeTimelogsWeekSplitByNewYearIsMerged(t *testing.T) {
	year2025 := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": [
			{"startDate": "2025-12-22", "endDate": "2025-12-28", "loggedTime": 300, "billableTime": 300,
			 "nonBillableTime": 0, "billedTime": 0, "estimatedTime": 0},
			{"startDate": "2025-12-29", "endDate": "2025-12-31", "loggedTime": 120, "billableTime": 60,
			 "nonBillableTime": 60, "billedTime": 60, "estimatedTime": 0}
		]}`)
	year2026 := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": [
			{"startDate": "2026-01-01", "endDate": "2026-01-04", "loggedTime": 180, "billableTime": 120,
			 "nonBillableTime": 60, "billedTime": 0, "estimatedTime": 0},
			{"startDate": "2026-01-05", "endDate": "2026-01-11", "loggedTime": 60, "billableTime": 0,
			 "nonBillableTime": 60, "billedTime": 0, "estimatedTime": 0}
		]}`)

	mcpServer := testutil.ProjectsMCPServerSequencedMock(t, http.StatusOK, year2025, year2026)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2025-12-22",
		"end_date":   "2026-01-11",
		"group_by":   "week",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Periods) != 3 {
			t.Fatalf("expected 3 weeks after merging the split one, got %d: %+v", len(res.Periods), res.Periods)
		}
		merged := res.Periods[1]
		if merged.StartDate != "2025-12-29" || merged.EndDate != "2026-01-04" {
			t.Errorf("expected the split week to span 2025-12-29 to 2026-01-04, got %s to %s",
				merged.StartDate, merged.EndDate)
		}
		// 120 + 180 logged, 60 + 120 billable, 60 billed → 120 unbilled billable.
		if merged.LoggedMinutes != 300 || merged.BillableMinutes != 180 || merged.UnbilledBillableMinutes != 120 {
			t.Errorf("merged week columns wrong: %+v", merged)
		}
		if res.Periods[0].EndDate != "2025-12-28" || res.Periods[2].StartDate != "2026-01-05" {
			t.Errorf("neighbouring weeks must be left alone: %+v", res.Periods)
		}
		if res.Totals.LoggedMinutes != 660 || res.Totals.GroupCount != 3 {
			t.Errorf("totals wrong: %+v", res.Totals)
		}
		assertReconciles(t, res)
	}))
}

func TestSummarizeTimelogsFullWeekEndingOnNewYearsEveIsNotMerged(t *testing.T) {
	year2025 := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": [
			{"startDate": "2025-12-25", "endDate": "2025-12-31", "loggedTime": 300, "billableTime": 300,
			 "nonBillableTime": 0, "billedTime": 0, "estimatedTime": 0}
		]}`)
	year2026 := []byte(`{"loggedTime": 0, "billableTime": 0, "nonBillableTime": 0, "billedTime": 0,
		"estimatedTime": 0, "dates": [
			{"startDate": "2026-01-01", "endDate": "2026-01-07", "loggedTime": 60, "billableTime": 0,
			 "nonBillableTime": 60, "billedTime": 0, "estimatedTime": 0}
		]}`)

	mcpServer := testutil.ProjectsMCPServerSequencedMock(t, http.StatusOK, year2025, year2026)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSummarizeTimelogs.String(), map[string]any{
		"start_date": "2025-12-25",
		"end_date":   "2026-01-07",
		"group_by":   "week",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		res := decodeSummary(t, result)

		if len(res.Periods) != 2 {
			t.Fatalf("expected 2 separate weeks, got %d: %+v", len(res.Periods), res.Periods)
		}
		if res.Periods[0].EndDate != "2025-12-31" || res.Periods[1].StartDate != "2026-01-01" {
			t.Errorf("full weeks must not be merged: %+v", res.Periods)
		}
		assertReconciles(t, res)
	}))
}
