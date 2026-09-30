package twprojects_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestUsersWorkload(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUsersWorkload.String(), map[string]any{
		"start_date":       "2023-01-01",
		"end_date":         "2023-01-31",
		"user_ids":         []float64{1, 2, 3},
		"user_company_ids": []float64{4, 5, 6},
		"user_team_ids":    []float64{7, 8, 9},
		"project_ids":      []float64{10, 11, 12},
		"page":             float64(1),
		"page_size":        float64(10),
	})
}

// TestUsersWorkloadIncludeTasksReachesTheWire pins the sideloads on the query
// string, since the endpoint answers without one it does not recognise.
func TestUsersWorkloadIncludeTasksReachesTheWire(t *testing.T) {
	tests := []struct {
		name         string
		includeTasks any
		want         []string
	}{
		{name: "omitted", want: []string{"users.workingHours.workingHoursEntry"}},
		{name: "false", includeTasks: false, want: []string{"users.workingHours.workingHoursEntry"}},
		{name: "true", includeTasks: true,
			want: []string{"users.workingHours.workingHoursEntry", "tasks.taskCapacities"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			args := map[string]any{"start_date": "2026-10-05", "end_date": "2026-10-09"}
			if tt.includeTasks != nil {
				args["include_tasks"] = tt.includeTasks
			}
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUsersWorkload.String(), args)

			if got := requestURL.Query()["include"]; !slices.Equal(got, tt.want) {
				t.Errorf("expected include %v, got %v", tt.want, got)
			}
		})
	}
}

// TestUsersWorkloadIncludeTasksTrimsEachTask pins that a sideloaded task comes
// back with the attributes that explain a day only, and that the trimmed shape
// still matches the published output schema.
func TestUsersWorkloadIncludeTasksTrimsEachTask(t *testing.T) {
	body := `{
		"workload": {"users": [{"userId": 456, "dates": {"2026-10-05": {
			"capacity": 104.2, "capacityMinutes": 500, "unavailableDay": false, "isHoliday": false,
			"projects": [{"capacity": 104.2, "capacityMinutes": 500,
				"project": {"id": 777, "type": "projects"}, "tasks": [{"id": 12345, "type": "tasks"}]}]
		}}}]},
		"meta": {"page": {"hasMore": false}},
		"included": {
			"tasks": {"12345": {"id": 12345, "name": "Example Task", "description": "long text",
				"priority": "high", "estimateMinutes": 600, "startDate": "2026-10-05", "dueDate": "2026-10-06",
				"assignees": [{"id": 456, "type": "users"}]}},
			"taskCapacities": {"1": {"id": 1, "taskId": 12345, "userId": 456, "date": "2026-10-05",
				"minutes": 500, "seconds": 0}}
		}
	}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(body))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUsersWorkload.String(), map[string]any{
		"start_date":    "2026-10-05",
		"end_date":      "2026-10-09",
		"include_tasks": true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		checkStructuredContentMatchesOutputSchema(twprojects.MethodUsersWorkload.String())(t, result)

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		text, ok := toolResult.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type: %T", toolResult.Content[0])
		}
		var decoded struct {
			Included struct {
				Tasks          map[string]map[string]any `json:"tasks"`
				TaskCapacities map[string]map[string]any `json:"taskCapacities"`
			} `json:"included"`
		}
		if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
			t.Fatalf("failed to decode the result: %s", err)
		}
		task, ok := decoded.Included.Tasks["12345"]
		if !ok {
			t.Fatalf("expected the task in the result, got %v", decoded.Included.Tasks)
		}
		for _, key := range []string{"id", "name", "startDate", "dueDate", "estimateMinutes", "assignees"} {
			if _, ok := task[key]; !ok {
				t.Errorf("expected %q on the task, got %v", key, task)
			}
		}
		for _, key := range []string{"description", "priority"} {
			if _, ok := task[key]; ok {
				t.Errorf("expected %q to be trimmed, got %v", key, task)
			}
		}
		if got := decoded.Included.TaskCapacities["1"]["minutes"]; got != float64(500) {
			t.Errorf("expected the split row in the result, got %v", decoded.Included.TaskCapacities)
		}
	}))
}
