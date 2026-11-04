package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestTasklistCreateBatchReachesTheWire pins one create per item, a project's
// lists in the order listed, and the new ID of each.
func TestTasklistCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/projects/20/", Status: http.StatusCreated, Body: []byte(`{"tasklistId":"202"}`)},
	}, http.StatusCreated, []byte(`{"tasklistId":"101"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistCreateBatch.String(), map[string]any{
		"tasklists": []any{
			map[string]any{"name": "First", "project_id": float64(10)},
			map[string]any{"name": "Other", "project_id": float64(20)},
			map[string]any{"name": "Second", "project_id": float64(10)},
			map[string]any{"name": "Third", "project_id": float64(10)},
		},
	}, expectBatchResult(false, "Created 4 of 4 tasklists", `item 2 "Other" → tasklist 202`,
		`item 1 "First" → tasklist 101`))

	var inProject10 []string
	for _, entry := range requestsOfMethod(*recorded, http.MethodPost) {
		if entry.URL.Path != "/projects/10/tasklists.json" {
			continue
		}
		for _, name := range []string{"First", "Second", "Third"} {
			if strings.Contains(string(entry.Body), `"`+name+`"`) {
				inProject10 = append(inProject10, name)
			}
		}
	}
	if strings.Join(inProject10, ",") != "First,Second,Third" {
		t.Errorf("expected project 10's lists in the order listed, got %v", inProject10)
	}
}

// TestTasklistCreateBatchRejectsAnInvalidItemBeforeWriting keeps an item with
// no project from leaving the rest half created.
func TestTasklistCreateBatchRejectsAnInvalidItemBeforeWriting(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"tasklistId":"1"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistCreateBatch.String(), map[string]any{
		"tasklists": []any{
			map[string]any{"name": "Fine", "project_id": float64(10)},
			map[string]any{"name": "Nowhere"},
		},
	}, expectBatchResult(true, "project_id"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestTasklistCreateBatchReportsEachFailure keeps going past a failed item.
func TestTasklistCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/projects/20/", Status: http.StatusForbidden, Body: []byte(`{}`)},
	}, http.StatusCreated, []byte(`{"tasklistId":"101"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistCreateBatch.String(), map[string]any{
		"tasklists": []any{
			map[string]any{"name": "A", "project_id": float64(20)},
			map[string]any{"name": "B", "project_id": float64(10)},
		},
	}, expectBatchResult(true, "Created 1 of 2 tasklists", `item 2 "B" → tasklist 101`, `Failed: item 1 "A"`))
}

// TestTasklistCreateBatchCapsEachProject keeps one project's sequential creates
// within the tool call's time.
func TestTasklistCreateBatchCapsEachProject(t *testing.T) {
	var tasklists []any
	for range 26 {
		tasklists = append(tasklists, map[string]any{"name": "List", "project_id": float64(10)})
	}
	tasklists = append(tasklists, map[string]any{"name": "Other", "project_id": float64(20)})
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"tasklistId":"1"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistCreateBatch.String(), map[string]any{
		"tasklists": tasklists,
	}, expectBatchResult(true, "Nothing was written", "project 10 has 26 tasklists, at most 25 per call"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}
