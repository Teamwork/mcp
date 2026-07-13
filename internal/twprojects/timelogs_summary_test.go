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

// assertReconciles asserts that the per-group columns sum to the totals block
// exactly, in minutes (minutes are the authoritative values).
func assertReconciles(t *testing.T, res tlResult) {
	t.Helper()

	var logged, billable, nonBillable, billed, unbilled int64
	for _, g := range res.Groups {
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
	if int64(len(res.Groups)) != res.Totals.GroupCount {
		t.Errorf("groupCount=%d but %d groups returned", res.Totals.GroupCount, len(res.Groups))
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
