package twprojects_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// allocationBatchItem returns the arguments of one create_allocations item.
func allocationBatchItem(title string, projectID float64) map[string]any {
	return map[string]any{
		"project_id":       projectID,
		"assigned_user_id": float64(456),
		"title":            title,
		"start_date":       "2026-09-01",
		"end_date":         "2026-09-30",
		"seconds_per_day":  float64(14400),
		"color":            "#3c8f7c",
	}
}

// allocationBatchPayload is the body the bulk create route receives.
type allocationBatchPayload struct {
	Allocations []map[string]any `json:"allocations"`
}

// TestAllocationCreateBatchSendsOneRequest pins the whole batch onto one bulk
// request, each item carrying its own fields, and the new IDs reported in
// order.
func TestAllocationCreateBatchSendsOneRequest(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/allocations/bulk.json",
		Status: http.StatusCreated,
		Body:   []byte(`{"allocations":[{"id":101},{"id":102,"overAllocated":true}]}`),
	}}, http.StatusNotFound, []byte(`{}`))

	review := allocationBatchItem("Review", 778)
	review["inform_of_over_allocation"] = false
	review["linked_task_ids"] = []any{float64(12345)}
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": []any{allocationBatchItem("Design", 777), review},
	}, expectBatchResult(false,
		"Created 2 allocations",
		`item 1 "Design" → allocation 101`,
		`item 2 "Review" → allocation 102`,
		`over their capacity for the period: item 2 "Review"`,
	))

	if len(*recorded) != 1 {
		t.Fatalf("expected one request for the whole batch, got %d", len(*recorded))
	}
	entry := (*recorded)[0]
	if entry.Method != http.MethodPost || entry.URL.Path != "/projects/api/v3/allocations/bulk.json" {
		t.Errorf("expected POST /projects/api/v3/allocations/bulk.json, got %s %s", entry.Method, entry.URL.Path)
	}
	var payload allocationBatchPayload
	if err := json.Unmarshal(entry.Body, &payload); err != nil {
		t.Fatalf("failed to decode body %q: %v", entry.Body, err)
	}
	if len(payload.Allocations) != 2 {
		t.Fatalf("expected 2 allocations in the body, got %d", len(payload.Allocations))
	}
	for i, want := range []struct {
		title   string
		project float64
		inform  bool
	}{
		{"Design", 777, true}, // defaulted on, as create_allocation does
		{"Review", 778, false},
	} {
		got := payload.Allocations[i]
		if got["title"] != want.title || got["projectId"] != want.project {
			t.Errorf("item %d: expected %q on project %v, got %v", i+1, want.title, want.project, got)
		}
		if got["assignedUserID"] != float64(456) || got["secondsPerDay"] != float64(14400) ||
			got["startedAt"] != "2026-09-01" || got["endedAt"] != "2026-09-30" || got["color"] != "#3c8f7c" {
			t.Errorf("item %d: required fields did not reach the wire: %v", i+1, got)
		}
		if got["informOfOverAllocation"] != want.inform {
			t.Errorf("item %d: expected informOfOverAllocation %v, got %v", i+1, want.inform,
				got["informOfOverAllocation"])
		}
	}
	if fmt.Sprint(payload.Allocations[1]["linkedTaskIDs"]) != "[12345]" {
		t.Errorf("expected linkedTaskIDs [12345] on item 2, got %v", payload.Allocations[1]["linkedTaskIDs"])
	}
}

// TestAllocationCreateBatchRejectedItem pins the all-or-nothing failure: the
// result says nothing was created and names the item the API rejected, by its
// position in the caller's list.
func TestAllocationCreateBatchRejectedItem(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/allocations/bulk.json",
		Status: http.StatusBadRequest,
		Body: []byte(`{"errors":[{"title":"invalid payload","detail":"start date after end date",` +
			`"meta":{"index":1}}]}`),
	}}, http.StatusNotFound, []byte(`{}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": []any{
			allocationBatchItem("Design", 777),
			allocationBatchItem("Build", 777),
			allocationBatchItem("Ship", 777),
		},
	}, expectBatchResult(true,
		"No allocations were created",
		`rejected item 2 "Build" (status 400): start date after end date`,
		"send the whole list again",
	))

	if len(*recorded) != 1 {
		t.Errorf("expected one request, got %d", len(*recorded))
	}
}

// TestAllocationCreateBatchRejectedBatch covers a rejection that names no item,
// such as a permission failure: still nothing was created.
func TestAllocationCreateBatchRejectedBatch(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/allocations/bulk.json",
		Status: http.StatusForbidden,
		Body:   []byte(`{"errors":[{"title":"forbidden action","detail":"access denied"}]}`),
	}}, http.StatusNotFound, []byte(`{}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": []any{allocationBatchItem("Design", 777)},
	}, expectBatchResult(true, "No allocations were created", "as a whole (status 403): access denied"))
}

// TestAllocationCreateBatchRejectsTooManyItems refuses an over-long list before
// anything is sent.
func TestAllocationCreateBatchRejectsTooManyItems(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"allocations":[]}`))

	items := make([]any, 51)
	for i := range items {
		items[i] = allocationBatchItem(fmt.Sprintf("Allocation %d", i+1), 777)
	}
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": items,
	}, expectBatchResult(true, "greater than 50"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestAllocationCreateBatchRejectsAnInvalidItemBeforeSending keeps a missing
// required field from costing the request: the endpoint would reject the whole
// batch for it. The item schema carries create_allocation's requirements, so
// the input validation refuses it before the handler runs.
func TestAllocationCreateBatchRejectsAnInvalidItemBeforeSending(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"allocations":[]}`))

	incomplete := allocationBatchItem("Build", 777)
	delete(incomplete, "color")
	delete(incomplete, "seconds_per_day")
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": []any{allocationBatchItem("Design", 777), incomplete},
	}, expectBatchResult(true, `missing properties: ["seconds_per_day" "color"]`))

	// A value only the handler can check, such as an unparseable date, is
	// reported per item, still before anything is sent.
	badDate := allocationBatchItem("Ship", 777)
	badDate["end_date"] = "end of September"
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreateBatch.String(), map[string]any{
		"allocations": []any{allocationBatchItem("Design", 777), badDate},
	}, expectBatchResult(true, "Nothing was written", "item 2: "))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}
