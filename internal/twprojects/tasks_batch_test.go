package twprojects_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// expectBatchResult asserts the result's error flag and that its text names
// every fragment, returning a check for ExecuteToolRequest.
func expectBatchResult(isError bool, fragments ...string) testutil.ExecuteToolRequestOption {
	return testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		var text string
		for _, content := range toolResult.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				text += textContent.Text
			}
		}
		if toolResult.IsError != isError {
			t.Errorf("expected IsError=%v, got %v: %s", isError, toolResult.IsError, text)
		}
		for _, fragment := range fragments {
			if !strings.Contains(text, fragment) {
				t.Errorf("expected the result to contain %q, got: %s", fragment, text)
			}
		}
	})
}

// TestTaskCreateBatchReachesTheWire pins one create per item, in the order
// listed, each carrying its own fields and the batch-wide notify flag.
func TestTaskCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/tasklists/10/", Status: http.StatusCreated, Body: []byte(`{"task":{"id":101}}`)},
		{Method: http.MethodPost, Match: "/tasklists/20/", Status: http.StatusCreated, Body: []byte(`{"task":{"id":202}}`)},
	}, http.StatusNotFound, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreateBatch.String(), map[string]any{
		"notify": false,
		"tasks": []any{
			map[string]any{"name": "Kickoff", "tasklist_id": float64(10), "due_date": "2026-10-01"},
			map[string]any{"name": "Review", "tasklist_id": float64(20), "priority": "high"},
		},
	}, expectBatchResult(false, "Created 2 of 2 tasks", `item 1 "Kickoff" → task 101`, `item 2 "Review" → task 202`))

	if len(*recorded) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(*recorded))
	}
	for i, want := range []struct {
		path, name string
	}{
		{"/projects/api/v3/tasklists/10/tasks.json", "Kickoff"},
		{"/projects/api/v3/tasklists/20/tasks.json", "Review"},
	} {
		entry := (*recorded)[i]
		if entry.URL.Path != want.path {
			t.Errorf("request %d: expected path %s, got %s", i, want.path, entry.URL.Path)
		}
		var payload struct {
			Task struct {
				Name     string  `json:"name"`
				DueAt    *string `json:"dueAt"`
				Priority *string `json:"priority"`
			} `json:"task"`
			Options struct {
				Notify bool `json:"notify"`
			} `json:"taskOptions"`
		}
		if err := json.Unmarshal(entry.Body, &payload); err != nil {
			t.Fatalf("request %d: failed to decode body %q: %v", i, entry.Body, err)
		}
		if payload.Task.Name != want.name {
			t.Errorf("request %d: expected name %q, got %q", i, want.name, payload.Task.Name)
		}
		if payload.Options.Notify {
			t.Errorf("request %d: expected notify false", i)
		}
	}
}

// TestTaskCreateBatchRejectsAnInvalidItemBeforeWriting keeps a bad item from
// leaving the rest of the batch half created.
func TestTaskCreateBatchRejectsAnInvalidItemBeforeWriting(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"task":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreateBatch.String(), map[string]any{
		"tasks": []any{
			map[string]any{"name": "Fine", "tasklist_id": float64(10)},
			map[string]any{"name": "Half placed", "tasklist_id": float64(10), "workflow_id": float64(3)},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2: workflow_id and stage_id"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestTaskCreateBatchReportsEachFailure keeps going past a failed item and
// names both what was created and what was not.
func TestTaskCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/tasklists/20/", Status: http.StatusForbidden, Body: []byte(`{}`)},
	}, http.StatusCreated, []byte(`{"task":{"id":101}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreateBatch.String(), map[string]any{
		"tasks": []any{
			map[string]any{"name": "A", "tasklist_id": float64(20)},
			map[string]any{"name": "B", "tasklist_id": float64(10)},
		},
	}, expectBatchResult(true, "Created 1 of 2 tasks", `item 2 "B" → task 101`, `Failed: item 1 "A"`))

	if len(*recorded) != 2 {
		t.Errorf("expected both items to be attempted, got %d requests", len(*recorded))
	}
}

// TestTaskUpdateBatchReachesTheWire pins one update per task, each carrying only
// its own changes.
func TestTaskUpdateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdateBatch.String(), map[string]any{
		"tasks": []any{
			map[string]any{"id": float64(1), "due_date": "2026-10-01"},
			map[string]any{"id": float64(2), "name": "Renamed"},
			map[string]any{"id": float64(3), "clear_parent_task": true},
		},
	}, expectBatchResult(false, "Updated 3 of 3 tasks"))

	writes := requestsOfMethod(*recorded, http.MethodPut)
	if len(writes) != 3 {
		t.Fatalf("expected 3 writes, got %d", len(writes))
	}
	bodies := make(map[string]string, len(writes))
	for _, entry := range writes {
		bodies[entry.URL.Path] = string(entry.Body)
	}
	paths := make([]string, 0, len(bodies))
	for path := range bodies {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	wantPaths := []string{
		"/projects/api/v3/tasks/1.json",
		"/projects/api/v3/tasks/2.json",
		"/projects/api/v3/tasks/3.json",
	}
	if !slices.Equal(paths, wantPaths) {
		t.Fatalf("expected paths %v, got %v", wantPaths, paths)
	}
	for path, want := range map[string]string{
		wantPaths[0]: `"dueAt":"2026-10-01"`,
		wantPaths[1]: `"name":"Renamed"`,
		wantPaths[2]: `"parentTaskId":0`,
	} {
		if !strings.Contains(bodies[path], want) {
			t.Errorf("%s: expected body to contain %s, got %s", path, want, bodies[path])
		}
	}
	if strings.Contains(bodies[wantPaths[0]], `"name"`) {
		t.Errorf("%s: expected no name, got %s", wantPaths[0], bodies[wantPaths[0]])
	}
}

// TestTaskUpdateBatchRejectsUnsafeItems covers the two items that would race:
// a task listed twice, and a move, which belongs to move_tasks.
func TestTaskUpdateBatchRejectsUnsafeItems(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdateBatch.String(), map[string]any{
		"tasks": []any{
			map[string]any{"id": float64(1), "name": "A"},
			map[string]any{"id": float64(1), "priority": "low"},
			map[string]any{"id": float64(2), "tasklist_id": float64(9)},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2: task 1 is listed more than once",
		"item 3: tasklist_id is not accepted", twprojects.MethodTaskMove.String()))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestTaskBatchItemsMirrorTheSingleTool keeps the batch item schemas in step
// with create_task and update_task, less the parameters the batch leaves out.
func TestTaskBatchItemsMirrorTheSingleTool(t *testing.T) {
	for _, tc := range []struct {
		batch, single toolWrapperFunc
		dropped       []string
	}{
		{twprojects.TaskCreateBatch, twprojects.TaskCreate, nil},
		{twprojects.TaskUpdateBatch, twprojects.TaskUpdate, []string{"tasklist_id"}},
	} {
		batch := tc.batch(nil).Tool
		single := tc.single(nil).Tool
		items := inputProperties(t, batch.InputSchema)["tasks"].Items
		for name := range inputProperties(t, single.InputSchema) {
			dropped := slices.Contains(tc.dropped, name) || slices.Contains([]string{
				"notify", "attachment_refs", "attachment_file_ids",
				"change_followers", "comment_followers", "complete_followers",
			}, name)
			if _, ok := items.Properties[name]; ok == dropped {
				t.Errorf("%s: item property %q present=%v, want %v", batch.Name, name, ok, !dropped)
			}
		}
	}
}

type toolWrapperFunc = func(*twapi.Engine) toolsets.ToolWrapper

func inputProperties(t *testing.T, inputSchema any) map[string]*jsonschema.Schema {
	t.Helper()
	schema, ok := inputSchema.(*jsonschema.Schema)
	if !ok {
		t.Fatalf("unexpected input schema type: %T", inputSchema)
	}
	return schema.Properties
}
