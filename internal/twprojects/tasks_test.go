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

func requestsOfMethod(recorded []testutil.ProjectsRecordedRequest, method string) []testutil.ProjectsRecordedRequest {
	var matched []testutil.ProjectsRecordedRequest
	for _, entry := range recorded {
		if entry.Method == method {
			matched = append(matched, entry)
		}
	}
	return matched
}

func TestTaskMoveUpdatesEachTaskOnce(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/tasks/2.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":2,"tasklist":{"id":10}}}`)},
		{Method: http.MethodGet, Match: "/tasks/5.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":5,"tasklist":{"id":10}}}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{2, 5},
		"tasklist_id": float64(20),
	})

	// v3 carries the subtree and drops an invalidated parent itself, so each
	// unrelated task costs exactly one write.
	writes := requestsOfMethod(*recorded, http.MethodPut)
	if len(writes) != 2 {
		t.Fatalf("expected 2 writes, got %d", len(writes))
	}
	for i, entry := range writes {
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
		if got := writes[i].URL.Path; got != want {
			t.Errorf("request %d: expected path %s, got %s", i, want, got)
		}
	}
}

// TestTaskMoveReadsNothingForASingleTask keeps the common case at one request:
// with one task named, nothing can be an ancestor of anything else.
func TestTaskMoveReadsNothingForASingleTask(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{2},
		"tasklist_id": float64(20),
	})

	if len(*recorded) != 1 {
		t.Fatalf("expected a single request, got %d", len(*recorded))
	}
	if (*recorded)[0].Method != http.MethodPut {
		t.Errorf("expected the only request to be the write, got %s", (*recorded)[0].Method)
	}
}

// TestTaskMoveSkipsDescendantListedFirst is the ordering hazard. The API detaches
// a task whose parent stays behind, so moving task 3 before its parent task 2
// would flatten the subtree instead of letting task 2 carry it.
func TestTaskMoveSkipsDescendantListedFirst(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/tasks/2.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":2,"tasklist":{"id":10}}}`)},
		{Method: http.MethodGet, Match: "/tasks/3.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":3,"tasklist":{"id":10},"parentTask":{"id":2}}}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{3, 2},
		"tasklist_id": float64(20),
	})

	writes := requestsOfMethod(*recorded, http.MethodPut)
	if len(writes) != 1 {
		t.Fatalf("expected only the ancestor to be written, got %d writes", len(writes))
	}
	if want := "/projects/api/v3/tasks/2.json"; writes[0].URL.Path != want {
		t.Errorf("expected the write to target %s, got %s", want, writes[0].URL.Path)
	}
}

// TestTaskMoveMovesChildOfTaskAlreadyInDestination guards the other half: a
// requested ancestor that is not moving carries nothing, so the child still has
// to move itself.
func TestTaskMoveMovesChildOfTaskAlreadyInDestination(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/tasks/2.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":2,"tasklist":{"id":20}}}`)},
		{Method: http.MethodGet, Match: "/tasks/3.json", Status: http.StatusOK,
			Body: []byte(`{"task":{"id":3,"tasklist":{"id":10},"parentTask":{"id":2}}}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    []float64{2, 3},
		"tasklist_id": float64(20),
	})

	writes := requestsOfMethod(*recorded, http.MethodPut)
	if len(writes) != 2 {
		t.Fatalf("expected both tasks to be written, got %d writes", len(writes))
	}
}

// TestTaskMoveRejectsOversizedList keeps the fan-out bounded: each task costs a
// read and a write, in sequence.
func TestTaskMoveRejectsOversizedList(t *testing.T) {
	ids := make([]float64, 51)
	for i := range ids {
		ids[i] = float64(i + 1)
	}
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskMove.String(), map[string]any{
		"task_ids":    ids,
		"tasklist_id": float64(20),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error when task_ids exceeds the cap")
		}
	}))
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

// TestTaskGetCarriesCompletedWork pins the pair of filters that decide whether a
// get answers with the task's completed subtasks and predecessors. The
// related-task filter reports active ones only, so a parent whose subtasks are
// all done came back with an empty subTaskIds — indistinguishable from a task
// that never had any, and with no parameter the caller could pass to tell them
// apart. includeCompletedPredecessors widens the whole related-task response
// despite its name, and cannot drop the row: the endpoint addresses the task by
// ID.
//
// Asserted on the query string, because the mock replies with the same canned
// body whether the filters are sent or not.
func TestTaskGetCarriesCompletedWork(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(), map[string]any{
		"id": float64(777),
	})

	for name, want := range map[string]string{
		"includeRelatedTasks":          "true",
		"includeCompletedPredecessors": "true",
	} {
		if got := lastURL.Query().Get(name); got != want {
			t.Errorf("expected %s=%s but got %q (raw query: %s)", name, want, got, lastURL.RawQuery)
		}
	}
}

// TestTaskSparseFieldsSubTaskIDsCarryRelatedTasks covers subTaskIds the way
// TestSparseFieldsPredecessorsCarryRelatedTasks covers predecessors: the API
// leaves it empty unless the request also asks for related tasks, and empty
// reads as "this task has no subtasks". Completed subtasks need the second flag
// on top, since the related-task filter reports active ones only.
func TestTaskSparseFieldsSubTaskIDsCarryRelatedTasks(t *testing.T) {
	for _, testCase := range []struct {
		method string
		args   map[string]any
	}{
		{method: twprojects.MethodTaskList.String()},
		{method: twprojects.MethodTaskGet.String(), args: map[string]any{"id": float64(777)}},
	} {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method,
				argsWithFields(testCase.args, "subTaskIds"))
			if got := lastURL.Query().Get("includeRelatedTasks"); got != "true" {
				t.Errorf("expected includeRelatedTasks=true alongside a subTaskIds selection but got %q "+
					"(raw query: %s)", got, lastURL.RawQuery)
			}

			// Control: the filter rides on the selection naming subTaskIds, not on
			// every selection, so an unrelated one must not carry it.
			mcpServer, lastURL = testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method,
				argsWithFields(testCase.args, "name"))
			if got := lastURL.Query().Get("includeRelatedTasks"); got != "" {
				t.Errorf("expected no includeRelatedTasks without a subTaskIds selection but got %q", got)
			}
		})
	}
}

// TestTaskGetSparseFieldsSubTaskIDsCarryCompletedWork is the get-only half of
// the above: a selection naming subTaskIds must still reach completed subtasks,
// which the related-task filter alone leaves out.
func TestTaskGetSparseFieldsSubTaskIDsCarryCompletedWork(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(),
		argsWithFields(map[string]any{"id": float64(777)}, "subTaskIds"))
	if got := lastURL.Query().Get("includeCompletedPredecessors"); got != "true" {
		t.Errorf("expected includeCompletedPredecessors=true alongside a subTaskIds selection but got %q "+
			"(raw query: %s)", got, lastURL.RawQuery)
	}

	// Control: a selection that needs no related tasks must not widen the get.
	mcpServer, lastURL = testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(),
		argsWithFields(map[string]any{"id": float64(777)}, "name"))
	if got := lastURL.Query().Get("includeCompletedPredecessors"); got != "" {
		t.Errorf("expected no includeCompletedPredecessors without a related-task selection but got %q", got)
	}
}

// TestTaskListShowCompletedReachesTheWire pins the flag a caller needs to find
// work that is already done. Scoped to a tasklist with a search term, a list
// that omits it answers zero rows for tasks that plainly exist, which is
// indistinguishable from no match.
//
// It governs the related-task lists a row carries too: a caller that asked to
// hide completed work would not expect completed IDs in `subTaskIds`. The SDK
// models that third filter as a plain bool, so "false" and "unset" both send
// nothing — only the true case reaches the wire.
func TestTaskListShowCompletedReachesTheWire(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		args     map[string]any
		expected map[string]string
	}{{
		name: "omitted keeps the API default",
		args: map[string]any{"tasklist_id": float64(777), "search_term": "design"},
		expected: map[string]string{
			"includeCompletedTasks":        "",
			"showCompletedLists":           "",
			"includeCompletedPredecessors": "",
		},
	}, {
		name: "true asks for completed work",
		args: map[string]any{
			"tasklist_id":    float64(777),
			"search_term":    "design",
			"show_completed": true,
		},
		expected: map[string]string{
			"includeCompletedTasks":        "true",
			"showCompletedLists":           "true",
			"includeCompletedPredecessors": "true",
		},
	}, {
		name: "false keeps it out",
		args: map[string]any{
			"tasklist_id":    float64(777),
			"search_term":    "design",
			"show_completed": false,
		},
		expected: map[string]string{
			"includeCompletedTasks":        "false",
			"showCompletedLists":           "false",
			"includeCompletedPredecessors": "",
		},
	}} {
		t.Run(testCase.name, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), testCase.args)

			for name, want := range testCase.expected {
				if got := lastURL.Query().Get(name); got != want {
					t.Errorf("expected %s=%q but got %q (raw query: %s)", name, want, got, lastURL.RawQuery)
				}
			}
		})
	}
}

// TestTaskCreateWorkflowPlacementReachesTheWire covers the whole point of the
// parameters: a task created straight into a stage saves the follow-up
// move_task_to_workflow_stage call, and that call is the one that used to be
// refused for non-administrators. The mocks answer the same body either way, so
// only the request shows whether the placement travelled.
func TestTaskCreateWorkflowPlacementReachesTheWire(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		want      *struct{ WorkflowID, StageID int64 }
	}{{
		name: "placed in a stage",
		arguments: map[string]any{
			"name":        "Test Task",
			"tasklist_id": float64(777),
			"workflow_id": float64(123),
			"stage_id":    float64(456),
		},
		want: &struct{ WorkflowID, StageID int64 }{123, 456},
	}, {
		// A "workflows" object on every unrelated create would be a payload
		// change for callers that never mention a workflow.
		name: "omitted when not asked for",
		arguments: map[string]any{
			"name":        "Test Task",
			"tasklist_id": float64(777),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
				[]byte(`{"task":{"id":12345}}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), tt.arguments)

			var payload struct {
				Task      map[string]any `json:"task"`
				Workflows *struct {
					WorkflowID int64 `json:"workflowId"`
					StageID    int64 `json:"stageId"`
				} `json:"workflows"`
			}
			if err := json.Unmarshal(*requestBody, &payload); err != nil {
				t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
			}

			// The endpoint reads "workflows" beside "task", never inside it.
			if _, ok := payload.Task["workflows"]; ok {
				t.Errorf("workflows must sit beside task, not inside it (body %q)", string(*requestBody))
			}

			switch {
			case tt.want == nil && payload.Workflows != nil:
				t.Errorf("expected no workflows block, got %+v (body %q)",
					*payload.Workflows, string(*requestBody))
			case tt.want != nil && payload.Workflows == nil:
				t.Fatalf("expected a workflows block (body %q)", string(*requestBody))
			case tt.want != nil:
				if payload.Workflows.WorkflowID != tt.want.WorkflowID {
					t.Errorf("expected workflowId %d, got %d", tt.want.WorkflowID, payload.Workflows.WorkflowID)
				}
				if payload.Workflows.StageID != tt.want.StageID {
					t.Errorf("expected stageId %d, got %d", tt.want.StageID, payload.Workflows.StageID)
				}
			}
		})
	}
}

// TestTaskCreateWorkflowPlacementNeedsBothIDs guards against the silent half of
// the endpoint's behaviour: a stage with no workflow is dropped, and the task
// lands in the backlog with a 201 that looks like success.
func TestTaskCreateWorkflowPlacementNeedsBothIDs(t *testing.T) {
	for _, arguments := range []map[string]any{
		{"name": "Test Task", "tasklist_id": float64(777), "stage_id": float64(456)},
		{"name": "Test Task", "tasklist_id": float64(777), "workflow_id": float64(123)},
	} {
		mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"task":{"id":12345}}`))
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), arguments,
			testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
				toolResult, ok := result.(*mcp.CallToolResult)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if !toolResult.IsError {
					t.Errorf("expected an error for %v, got success", arguments)
				}
			}),
		)
	}
}
