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

const taskSplitRows = `{"capacities":[` +
	`{"id":1,"taskId":12345,"userId":456,"date":"2026-10-05","minutes":90,"seconds":0},` +
	`{"id":2,"taskId":12345,"userId":456,"date":"2026-10-06","minutes":30,"seconds":0}]}`

// taskSplitResult runs a tool call and hands back its result, whatever it is.
func taskSplitResult(
	t *testing.T,
	mcpServer *mcp.Server,
	method string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	var toolResult *mcp.CallToolResult
	testutil.ExecuteToolRequest(t, mcpServer, method, args,
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			var ok bool
			if toolResult, ok = result.(*mcp.CallToolResult); !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
		}))
	return toolResult
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	return text.Text
}

// TestTaskSplitSetReachesTheWire pins the verbs a split write sends, because
// the create and replace routes share one path, and the replace route is the
// one that answers 404 for a user who has no split yet.
func TestTaskSplitSetReachesTheWire(t *testing.T) {
	args := map[string]any{
		"task_id": float64(12345),
		"user_id": float64(456),
		"dates": []any{
			map[string]any{"date": "2026-10-05", "minutes": float64(90)},
			map[string]any{"date": "2026-10-06", "minutes": float64(30)},
		},
	}
	const wantBody = `{"taskCapacity":{"userId":456,"dates":[` +
		`{"date":"2026-10-05","minutes":90},{"date":"2026-10-06","minutes":30}]}}`

	tests := []struct {
		name        string
		routes      []testutil.ProjectsMockRoute
		wantMethods []string
		wantError   bool
		wantText    string
	}{{
		name: "replaces an existing split",
		routes: []testutil.ProjectsMockRoute{
			{Match: "/tasks/12345/capacity", Method: http.MethodPut, Status: http.StatusOK, Body: []byte(taskSplitRows)},
		},
		wantMethods: []string{http.MethodPut},
		wantText:    `"totalMinutes":120`,
	}, {
		name: "creates the first split",
		routes: []testutil.ProjectsMockRoute{
			{Match: "/tasks/12345/capacity", Method: http.MethodPut, Status: http.StatusNotFound, Body: []byte(`{}`)},
			{Match: "/tasks/12345/capacity", Method: http.MethodPost, Status: http.StatusCreated,
				Body: []byte(taskSplitRows)},
		},
		wantMethods: []string{http.MethodPut, http.MethodPost},
		wantText:    `"totalMinutes":120`,
	}, {
		name: "an even split is not stored",
		routes: []testutil.ProjectsMockRoute{
			{Match: "/tasks/12345/capacity", Method: http.MethodPut, Status: http.StatusNotFound, Body: []byte(`{}`)},
			{Match: "/tasks/12345/capacity", Method: http.MethodPost, Status: http.StatusNoContent},
		},
		wantMethods: []string{http.MethodPut, http.MethodPost},
		wantText:    "even spread",
	}, {
		name: "a rejected split is not retried as a create",
		routes: []testutil.ProjectsMockRoute{
			{Match: "/tasks/12345/capacity", Method: http.MethodPut, Status: http.StatusBadRequest,
				Body: []byte(`{"errors":[{"detail":"capacity date is outside the task's date range"}]}`)},
		},
		wantMethods: []string{http.MethodPut},
		wantError:   true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, tt.routes, http.StatusOK, []byte(`{}`))
			result := taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitSet.String(), args)

			if result.IsError != tt.wantError {
				t.Fatalf("expected IsError %t, got %t: %s", tt.wantError, result.IsError, resultText(t, result))
			}
			if len(*recorded) != len(tt.wantMethods) {
				t.Fatalf("expected %d requests, got %d", len(tt.wantMethods), len(*recorded))
			}
			for i, request := range *recorded {
				if request.Method != tt.wantMethods[i] {
					t.Errorf("request %d: expected %s, got %s", i, tt.wantMethods[i], request.Method)
				}
				if request.URL.Path != "/projects/api/v3/tasks/12345/capacity.json" {
					t.Errorf("request %d: unexpected path %s", i, request.URL.Path)
				}
				if got := strings.TrimSpace(string(request.Body)); got != wantBody {
					t.Errorf("request %d: expected body %s, got %s", i, wantBody, got)
				}
			}
			if tt.wantText != "" && !strings.Contains(resultText(t, result), tt.wantText) {
				t.Errorf("expected %q in the result, got %s", tt.wantText, resultText(t, result))
			}
		})
	}
}

func TestTaskSplitSetClearDeletesTheUsersSplit(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "/tasks/12345/capacity", Method: http.MethodDelete, Status: http.StatusNoContent},
	}, http.StatusOK, []byte(`{}`))
	result := taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitSet.String(), map[string]any{
		"task_id": float64(12345),
		"user_id": float64(456),
		"clear":   true,
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	if len(*recorded) != 1 || (*recorded)[0].Method != http.MethodDelete {
		t.Fatalf("expected a single DELETE, got %+v", *recorded)
	}
	// an empty list would clear every assignee's split, not just this one's
	if got := strings.TrimSpace(string((*recorded)[0].Body)); got != `{"userIds":[456]}` {
		t.Errorf("expected the delete to name the user, got %s", got)
	}
}

func TestTaskSplitSetRejectsInvalidInput(t *testing.T) {
	day := map[string]any{"date": "2026-10-05", "minutes": float64(60)}
	tests := []struct {
		name string
		args map[string]any
	}{{
		name: "neither dates nor clear",
		args: map[string]any{},
	}, {
		name: "both dates and clear",
		args: map[string]any{"dates": []any{day}, "clear": true},
	}, {
		name: "a date twice",
		args: map[string]any{"dates": []any{day, day}},
	}, {
		name: "a timestamp instead of a date",
		args: map[string]any{"dates": []any{map[string]any{"date": "2026-10-05T09:00:00Z", "minutes": float64(60)}}},
	}, {
		name: "fractional minutes",
		args: map[string]any{"dates": []any{map[string]any{"date": "2026-10-05", "minutes": 1.5}}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
			tt.args["task_id"] = float64(12345)
			tt.args["user_id"] = float64(456)
			result := taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitSet.String(), tt.args)

			if !result.IsError {
				t.Error("expected an error tool result")
			}
			if len(*recorded) != 0 {
				t.Errorf("expected nothing sent, got %d requests", len(*recorded))
			}
		})
	}
}

// TestTaskSplitGetNamesEveryAssignee pins that an omitted user_ids reads the
// task's user assignees first: the list endpoint defaults to the caller's own
// rows, so a split held by anyone else would otherwise read as absent.
func TestTaskSplitGetNamesEveryAssignee(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "/tasks/capacity", Status: http.StatusOK, Body: []byte(taskSplitRows)},
		{Match: "/tasks/12345", Status: http.StatusOK, Body: []byte(`{"task":{"id":12345,"assignees":[` +
			`{"id":456,"type":"users"},{"id":789,"type":"users"},{"id":3,"type":"teams"}]}}`)},
	}, http.StatusOK, []byte(`{}`))
	result := taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitGet.String(), map[string]any{
		"task_id": float64(12345),
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	if len(*recorded) != 2 {
		t.Fatalf("expected a task read then a split list, got %d requests", len(*recorded))
	}
	if got := (*recorded)[0].URL.Query().Get("fields[tasks]"); got != "assignees,id" && got != "assignees" {
		t.Errorf("expected the task read to select assignees, got %q", got)
	}
	query := (*recorded)[1].URL.Query()
	if got := query.Get("userIds"); got != "456,789" {
		t.Errorf("expected the user assignees only, got userIds=%q", got)
	}
	if got := query.Get("taskId"); got != "12345" {
		t.Errorf("expected taskId=12345, got %q", got)
	}

	var split struct {
		Capacities []json.RawMessage `json:"capacities"`
	}
	if err := json.Unmarshal([]byte(resultText(t, result)), &split); err != nil {
		t.Fatalf("failed to decode the result: %s", err)
	}
	if len(split.Capacities) != 2 {
		t.Errorf("expected 2 rows, got %d", len(split.Capacities))
	}
}

func TestTaskSplitGetNamedUsersSkipTheTaskRead(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "/tasks/capacity", Status: http.StatusOK, Body: []byte(taskSplitRows)},
	}, http.StatusOK, []byte(`{}`))
	taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitGet.String(), map[string]any{
		"task_id":  float64(12345),
		"user_ids": []any{float64(456)},
	})

	if len(*recorded) != 1 {
		t.Fatalf("expected a single split list, got %d requests", len(*recorded))
	}
	if got := (*recorded)[0].URL.Query().Get("userIds"); got != "456" {
		t.Errorf("expected userIds=456, got %q", got)
	}
}

// TestTaskSplitGetStopsPaging pins the page bound, so an endpoint that keeps
// reporting more pages cannot hold the call open.
func TestTaskSplitGetStopsPaging(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "/tasks/capacity", Status: http.StatusOK,
			Body: []byte(`{"capacities":[],"meta":{"page":{"hasMore":true}}}`)},
	}, http.StatusOK, []byte(`{}`))
	taskSplitResult(t, mcpServer, twprojects.MethodTaskSplitGet.String(), map[string]any{
		"task_id":  float64(12345),
		"user_ids": []any{float64(456)},
	})

	if len(*recorded) != 20 {
		t.Errorf("expected paging to stop at 20 requests, got %d", len(*recorded))
	}
}
