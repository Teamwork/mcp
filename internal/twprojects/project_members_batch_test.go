package twprojects_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestProjectMemberAddBatchReachesTheWire pins one PUT per project, each with
// its own users.
func TestProjectMemberAddBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectMemberAddBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"project_id": float64(10), "user_ids": []any{float64(1)}},
			map[string]any{"project_id": float64(20), "user_ids": []any{float64(2), float64(3)}},
		},
	}, expectBatchResult(false, "Added users to 2 of 2 projects"))

	want := map[string][]int64{
		"/projects/api/v3/projects/10/people.json": {1},
		"/projects/api/v3/projects/20/people.json": {2, 3},
	}
	writes := requestsOfMethod(*recorded, http.MethodPut)
	if len(writes) != len(want) {
		t.Fatalf("expected %d writes, got %d", len(want), len(writes))
	}
	for _, entry := range writes {
		var payload struct {
			UserIDs []int64 `json:"userIds"`
		}
		if err := json.Unmarshal(entry.Body, &payload); err != nil {
			t.Fatalf("failed to decode body %q: %v", entry.Body, err)
		}
		if !slices.Equal(payload.UserIDs, want[entry.URL.Path]) {
			t.Errorf("%s: expected users %v, got %v", entry.URL.Path, want[entry.URL.Path], payload.UserIDs)
		}
	}
}

// TestProjectMemberAddBatchRejectsAnInvalidItemBeforeWriting covers an item
// naming no users and a repeated project.
func TestProjectMemberAddBatchRejectsAnInvalidItemBeforeWriting(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectMemberAddBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"project_id": float64(10), "user_ids": []any{float64(1)}},
			map[string]any{"project_id": float64(10), "user_ids": []any{float64(2)}},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2: project 10 is listed more than once"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestProjectMemberAddBatchReportsEachFailure names the project the caller is
// not allowed to add people to.
func TestProjectMemberAddBatchReportsEachFailure(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPut, Match: "/projects/20/", Status: http.StatusForbidden, Body: []byte(`{}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectMemberAddBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"project_id": float64(10), "user_ids": []any{float64(1)}},
			map[string]any{"project_id": float64(20), "user_ids": []any{float64(1)}},
		},
	}, expectBatchResult(true, "Added users to 1 of 2 projects", "Failed: project 20"))
}
