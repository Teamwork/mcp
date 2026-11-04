package twprojects_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestProjectUpdateBatchReachesTheWire pins one PUT per project, each carrying
// only its own changes.
func TestProjectUpdateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectUpdateBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"id": float64(1), "status": "archived"},
			map[string]any{"id": float64(2), "tag_ids": []any{float64(9)}},
		},
	}, expectBatchResult(false, "Updated 2 of 2 projects"))

	want := []string{"/projects/1.json", "/projects/2.json"}
	if got := recordedPaths(*recorded, http.MethodPut); !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for _, entry := range requestsOfMethod(*recorded, http.MethodPut) {
		body := string(entry.Body)
		archived := strings.Contains(body, `"archived"`)
		if (entry.URL.Path == "/projects/1.json") != archived {
			t.Errorf("%s: unexpected body %s", entry.URL.Path, body)
		}
	}
}

// TestProjectUpdateBatchRejectsAnInvalidItemBeforeWriting covers an unknown
// status and a repeated project.
func TestProjectUpdateBatchRejectsAnInvalidItemBeforeWriting(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectUpdateBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"id": float64(1), "name": "A"},
			map[string]any{"id": float64(1), "status": "archived"},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2: project 1 is listed more than once"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestProjectUpdateBatchReportsEachFailure lists a project the caller cannot
// edit without dropping the others.
func TestProjectUpdateBatchReportsEachFailure(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPut, Match: "/projects/2.json", Status: http.StatusForbidden, Body: []byte(`{}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectUpdateBatch.String(), map[string]any{
		"projects": []any{
			map[string]any{"id": float64(1), "status": "archived"},
			map[string]any{"id": float64(2), "status": "archived"},
		},
	}, expectBatchResult(true, "Updated 1 of 2 projects", "Failed: project 2"))
}
