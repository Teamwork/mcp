package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTaskCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"task":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), map[string]any{
		"name":              "Example",
		"tasklist_id":       float64(123),
		"description":       "This is an example task.",
		"priority":          "high",
		"progress":          float64(50),
		"start_date":        "2023-10-01",
		"due_date":          "2023-10-15",
		"estimated_minutes": float64(120),
		"parent_task_id":    float64(456),
		"assignees": map[string]any{
			"user_ids":     []float64{1, 2, 3},
			"team_ids":     []float64{4, 5},
			"company_ids":  []float64{6, 7},
			"job_role_ids": []float64{8, 9},
		},
		"tag_ids": []float64{1, 2, 3},
		"predecessors": []map[string]any{
			{
				"task_id": float64(456),
				"type":    "start",
			},
			{
				"task_id": float64(789),
				"type":    "complete",
			},
		},
	})
}

func TestTaskUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":                float64(123),
		"name":              "Example",
		"tasklist_id":       float64(123),
		"description":       "This is an example task.",
		"priority":          "high",
		"progress":          float64(50),
		"start_date":        "2023-10-01",
		"due_date":          "2023-10-15",
		"estimated_minutes": float64(120),
		"parent_task_id":    float64(456),
		"assignees": map[string]any{
			"user_ids":     []float64{1, 2, 3},
			"team_ids":     []float64{4, 5},
			"company_ids":  []float64{6, 7},
			"job_role_ids": []float64{8, 9},
		},
		"tag_ids": []float64{1, 2, 3},
		"predecessors": []map[string]any{
			{
				"task_id": float64(456),
				"type":    "start",
			},
			{
				"task_id": float64(789),
				"type":    "complete",
			},
		},
	})
}

func TestTaskUpdateClearAssignees(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":              float64(123),
		"clear_assignees": true,
	})

	var payload struct {
		Task struct {
			Assignees struct {
				UserIDs    []int64 `json:"userIds"`
				CompanyIDs []int64 `json:"companyIds"`
				TeamIDs    []int64 `json:"teamIds"`
				JobRoleIDs []int64 `json:"jobRoleIds"`
			} `json:"assignees"`
		} `json:"task"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}

	assignees := payload.Task.Assignees
	// Empty (non-null) arrays are what the API expects to unassign a task; a
	// null value would leave the dimension unchanged.
	if assignees.UserIDs == nil || assignees.CompanyIDs == nil ||
		assignees.TeamIDs == nil || assignees.JobRoleIDs == nil {
		t.Fatalf("expected empty (non-null) assignee arrays, got body %q", string(*requestBody))
	}
	if len(assignees.UserIDs) != 0 || len(assignees.CompanyIDs) != 0 ||
		len(assignees.TeamIDs) != 0 || len(assignees.JobRoleIDs) != 0 {
		t.Errorf("expected all assignee arrays to be empty, got %+v", assignees)
	}
}

func TestTaskUpdateAssignJobRoles(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id": float64(123),
		"assignees": map[string]any{
			"job_role_ids": []float64{8, 9},
		},
	})

	var payload struct {
		Task struct {
			Assignees struct {
				JobRoleIDs []int64 `json:"jobRoleIds"`
			} `json:"assignees"`
		} `json:"task"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}
	if got := payload.Task.Assignees.JobRoleIDs; len(got) != 2 || got[0] != 8 || got[1] != 9 {
		t.Errorf("expected jobRoleIds [8 9], got %v (body %q)", got, string(*requestBody))
	}
}

func TestTaskUpdateClearAssigneesConflictsWithAssignees(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":              float64(123),
		"clear_assignees": true,
		"assignees": map[string]any{
			"user_ids": []float64{1, 2, 3},
		},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error when combining clear_assignees with a non-empty assignees value")
		}
	}))
}

func TestTaskUpdateClearParentTask(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":                float64(123),
		"clear_parent_task": true,
	})

	var payload struct {
		Task struct {
			ParentTaskID *int64 `json:"parentTaskId"`
		} `json:"task"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}

	// Zero is the sentinel the v3 API accepts to detach a subtask. Omitting the
	// field entirely leaves the parent link intact, which is the silent no-op
	// this parameter exists to avoid.
	if payload.Task.ParentTaskID == nil {
		t.Fatalf("expected parentTaskId to reach the wire, got body %q", string(*requestBody))
	}
	if *payload.Task.ParentTaskID != 0 {
		t.Errorf("expected parentTaskId 0, got %d (body %q)", *payload.Task.ParentTaskID, string(*requestBody))
	}
}

func TestTaskUpdateClearParentTaskConflictsWithParentTaskID(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":                float64(123),
		"clear_parent_task": true,
		"parent_task_id":    float64(456),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error when combining clear_parent_task with parent_task_id")
		}
	}))
}

// TestTaskUpdateNullParentTaskIDIsNotAClear pins the meaning of an explicit
// null. Every optional parameter advertises null so OpenAI strict-mode clients,
// which must send every property, can fill the unset ones (see commit be42c41).
// Treating null as "detach from parent" would silently promote a subtask to
// top-level on any unrelated update those clients make.
func TestTaskUpdateNullParentTaskIDIsNotAClear(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":             float64(123),
		"parent_task_id": nil,
		"name":           "renamed",
	})

	var payload struct {
		Task map[string]any `json:"task"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}
	if _, ok := payload.Task["parentTaskId"]; ok {
		t.Errorf("expected parentTaskId to be omitted for a null value, got body %q", string(*requestBody))
	}
}

func TestTaskMoveUpdatesEachTaskOnce(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{2, 5},
		"tasklist_id": float64(20),
	})

	// v3 carries the subtree and drops an invalidated parent itself, so each
	// requested task costs exactly one write and nothing else.
	if len(*recorded) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(*recorded))
	}
	for i, entry := range *recorded {
		if entry.Method != http.MethodPut {
			t.Errorf("request %d: expected PUT, got %s", i, entry.Method)
		}
		var payload struct {
			Task struct {
				TasklistID   *int64 `json:"tasklistId"`
				ParentTaskID *int64 `json:"parentTaskId"`
			} `json:"task"`
		}
		if err := json.Unmarshal(entry.Body, &payload); err != nil {
			t.Fatalf("request %d: failed to decode body %q: %v", i, string(entry.Body), err)
		}
		if payload.Task.TasklistID == nil || *payload.Task.TasklistID != 20 {
			t.Errorf("request %d: expected tasklistId 20, got %v", i, payload.Task.TasklistID)
		}
		// Sending a parent would make the API validate it instead of dropping the
		// inherited one, which is what re-introduces "parent task is in another
		// list".
		if payload.Task.ParentTaskID != nil {
			t.Errorf("request %d: expected no parentTaskId, got %d", i, *payload.Task.ParentTaskID)
		}
	}

	for i, want := range []string{"/projects/api/v3/tasks/2.json", "/projects/api/v3/tasks/5.json"} {
		if got := (*recorded)[i].Path; got != want {
			t.Errorf("request %d: expected path %s, got %s", i, want, got)
		}
	}
}

func TestTaskMoveDeduplicatesTaskIDs(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{2, 2, 2},
		"tasklist_id": float64(20),
	})

	if len(*recorded) != 1 {
		t.Fatalf("expected a repeated ID to be written once, got %d requests", len(*recorded))
	}
}

func TestTaskMoveRequiresTaskIDs(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{},
		"tasklist_id": float64(20),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error when task_ids is empty")
		}
	}))
}

func TestTaskDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTaskComplete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskComplete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTaskGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTaskList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term":         "test",
		"tag_ids":             []float64{1, 2, 3},
		"match_all_tags":      true,
		"assignee_user_ids":   []float64{4, 5, 6},
		"created_after":       "2023-10-01T00:00:00Z",
		"created_before":      "2023-10-31T23:59:59Z",
		"created_by_user_ids": []float64{7, 8, 9},
		"updated_after":       "2023-10-01T00:00:00Z",
		"updated_before":      "2023-10-31T23:59:59Z",
		"completed_after":     "2023-10-01T00:00:00Z",
		"completed_before":    "2023-10-31T23:59:59Z",
		"due_after":           "2023-10-01",
		"due_before":          "2023-10-31",
		"show_completed":      true,
		"only_unassigned":     true,
		"only_unplanned":      true,
		"page":                float64(1),
		"page_size":           float64(10),
	})
}

func TestTaskListByTasklist(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"tasklist_id":         float64(123),
		"search_term":         "test",
		"tag_ids":             []float64{1, 2, 3},
		"match_all_tags":      true,
		"assignee_user_ids":   []float64{4, 5, 6},
		"created_after":       "2023-10-01T00:00:00Z",
		"created_before":      "2023-10-31T23:59:59Z",
		"created_by_user_ids": []float64{7, 8, 9},
		"updated_after":       "2023-10-01T00:00:00Z",
		"updated_before":      "2023-10-31T23:59:59Z",
		"completed_after":     "2023-10-01T00:00:00Z",
		"completed_before":    "2023-10-31T23:59:59Z",
		"due_after":           "2023-10-01",
		"due_before":          "2023-10-31",
		"show_completed":      true,
		"only_unassigned":     true,
		"only_unplanned":      true,
		"page":                float64(1),
		"page_size":           float64(10),
	})
}

func TestTaskListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"project_id":          float64(123),
		"search_term":         "test",
		"assignee_user_ids":   []float64{4, 5, 6},
		"tag_ids":             []float64{1, 2, 3},
		"match_all_tags":      true,
		"created_after":       "2023-10-01T00:00:00Z",
		"created_before":      "2023-10-31T23:59:59Z",
		"created_by_user_ids": []float64{7, 8, 9},
		"updated_after":       "2023-10-01T00:00:00Z",
		"updated_before":      "2023-10-31T23:59:59Z",
		"completed_after":     "2023-10-01T00:00:00Z",
		"completed_before":    "2023-10-31T23:59:59Z",
		"due_after":           "2023-10-01",
		"due_before":          "2023-10-31",
		"show_completed":      true,
		"only_unassigned":     true,
		"only_unplanned":      true,
		"page":                float64(1),
		"page_size":           float64(10),
	})
}
