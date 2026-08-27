package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestMilestoneCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"milestoneId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneCreate.String(), map[string]any{
		"name":        "Example",
		"project_id":  float64(123),
		"description": "Example milestone description",
		"due_date":    "20231231",
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
		"tasklist_ids": []float64{8, 9},
		"tag_ids":      []float64{10, 11, 12},
	})
}

func TestMilestoneUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "Example milestone description",
		"due_date":    "20231231",
		"assignees": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
		"tasklist_ids": []float64{8, 9},
		"tag_ids":      []float64{10, 11, 12},
	})
}

func TestMilestoneDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMilestoneGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMilestoneList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"due_after":      "2026-03-04",
		"due_before":     "2026-05-06",
		"show_completed": false,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}

func TestMilestoneListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneList.String(), map[string]any{
		"project_id":     float64(123),
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}

// TestMilestoneListDeadlineFilters covers the reason these filters exist: a
// schedule check bounds the deadline server-side instead of paging every
// milestone and discarding most of them, so the dates have to reach the wire.
func TestMilestoneListDeadlineFilters(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneList.String(), map[string]any{
		"due_after":  "2026-03-04",
		"due_before": "2026-05-06",
	})

	if len(*recorded) == 0 {
		t.Fatal("expected the milestones to be listed")
	}
	query := (*recorded)[0].URL.Query()
	if got := query.Get("dueAfter"); got != "2026-03-04" {
		t.Errorf("expected dueAfter=2026-03-04 but got %q", got)
	}
	if got := query.Get("dueBefore"); got != "2026-05-06" {
		t.Errorf("expected dueBefore=2026-05-06 but got %q", got)
	}
}

// TestMilestoneListShowCompleted pins the one parameter that is not passed
// straight through. Milestones are returned in every state, so hiding the
// completed ones means naming the incomplete ones, and only an explicit false
// may do that: leaving it out has to keep the endpoint's own behaviour.
func TestMilestoneListShowCompleted(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		want      string
	}{{
		name:      "false narrows the list to incomplete milestones",
		arguments: map[string]any{"show_completed": false},
		want:      "incomplete",
	}, {
		name:      "true leaves every state in",
		arguments: map[string]any{"show_completed": true},
		want:      "",
	}, {
		name:      "unset leaves every state in",
		arguments: map[string]any{},
		want:      "",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMilestoneList.String(), tt.arguments)

			if len(*recorded) == 0 {
				t.Fatal("expected the milestones to be listed")
			}
			if got := (*recorded)[0].URL.Query().Get("status"); got != tt.want {
				t.Errorf("expected status=%q but got %q", tt.want, got)
			}
		})
	}
}

// TestMilestoneAssigneesRejectJobRoles pins both halves of the job-role gap. The
// milestone endpoints have no job-role assignee: create answers a job-role-only
// list with 422 "Invalid milestone assignees", and update accepts it, answers 200
// and leaves the assignees as they were — so a caller is told an assignment
// happened when none did. A job role sent alongside a user is dropped from both
// without a word. The schema therefore must not offer job_role_ids, and the
// handler must reject one anyway, since the SDK validates the schema and a schema
// with no additionalProperties: false still lets the property through.
func TestMilestoneAssigneesRejectJobRoles(t *testing.T) {
	methods := []string{
		twprojects.MethodMilestoneCreate.String(),
		twprojects.MethodMilestoneUpdate.String(),
	}

	t.Run("schema does not advertise job_role_ids", func(t *testing.T) {
		schemas := toolInputSchemas(t)
		for _, method := range methods {
			schema, ok := schemas[method]
			if !ok {
				t.Fatalf("%s is not registered", method)
			}
			assignees, ok := schema.Properties["assignees"]
			if !ok {
				t.Fatalf("%s has no assignees parameter", method)
			}
			// The update tool wraps the object in AnyOf with null.
			object := assignees
			if object.Type != "object" {
				for _, branch := range assignees.AnyOf {
					if branch.Type == "object" {
						object = branch
						break
					}
				}
			}
			if object.Type != "object" {
				t.Fatalf("%s assignees has no object branch", method)
			}
			if _, ok := object.Properties["job_role_ids"]; ok {
				t.Errorf("%s advertises assignees.job_role_ids, which the endpoint refuses", method)
			}
			if _, ok := object.Properties["user_ids"]; !ok {
				t.Errorf("%s dropped assignees.user_ids", method)
			}
		}
	})

	t.Run("handler rejects a job role sent anyway", func(t *testing.T) {
		for _, method := range methods {
			mcpServer, requestBody := testutil.ProjectsMCPServerMockWithRequestBody(t,
				http.StatusCreated, []byte(`{"milestoneId":"123"}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"name":       "Example",
				"id":         float64(123),
				"project_id": float64(123),
				"due_date":   "20231231",
				"assignees": map[string]any{
					"user_ids":     []float64{1},
					"job_role_ids": []float64{2},
				},
			}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
				t.Helper()

				toolResult, ok := result.(*mcp.CallToolResult)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if !toolResult.IsError {
					t.Fatalf("%s accepted a job-role assignee", method)
				}
				textContent, ok := toolResult.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatalf("unexpected content type: %T", toolResult.Content[0])
				}
				if !strings.Contains(textContent.Text, "job roles") {
					t.Errorf("%s error does not name job roles: %q", method, textContent.Text)
				}
			}))
			// Rejected locally, so nothing is sent: a request here means the user
			// half was applied while the job role was silently dropped.
			if len(*requestBody) > 0 {
				t.Errorf("%s sent a request despite the job-role assignee: %s", method, *requestBody)
			}
		}
	})
}
