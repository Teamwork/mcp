package twprojects_test

import (
	"net/http"
	"testing"

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
