package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTimelogCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timelog":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreate.String(), map[string]any{
		"description": "Example timelog description",
		"date":        "2023-12-31",
		"time":        "12:00:00",
		"is_utc":      true,
		"hours":       float64(1),
		"minutes":     float64(30),
		"billable":    true,
		"project_id":  float64(123),
		"task_id":     float64(456),
		"user_id":     float64(789),
		"tag_ids":     []float64{10, 11, 12},
	})
}

func TestTimelogUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogUpdate.String(), map[string]any{
		"id":          float64(123),
		"description": "Example timelog description",
		"date":        "2023-12-31",
		"time":        "12:00:00",
		"is_utc":      true,
		"hours":       float64(1),
		"minutes":     float64(30),
		"billable":    true,
		"project_id":  float64(123),
		"task_id":     float64(456),
		"user_id":     float64(789),
		"tag_ids":     []float64{10, 11, 12},
	})
}

func TestTimelogDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimelogGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimelogList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogList.String(), map[string]any{
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	})
}

// TestTimelogListAcceptsPlainDates pins the exact arguments that used to fail.
// A plain YYYY-MM-DD is what a model emits when asked for a date range, so the
// binder rejecting it broke the first call of every time-reporting conversation.
// The window has to reach the end of the closing day: resolving end_date to its
// first instant would answer the question while silently omitting that day's
// timelogs.
func TestTimelogListAcceptsPlainDates(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogList.String(), map[string]any{
		"start_date": "2026-06-24",
		"end_date":   "2026-08-03",
	})

	query := lastURL.Query()
	if got := query.Get("startDate"); !strings.HasPrefix(got, "2026-06-24T00:00:00") {
		t.Errorf("expected startDate at the start of 2026-06-24 but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
	if got := query.Get("endDate"); !strings.HasPrefix(got, "2026-08-03T23:59:59") {
		t.Errorf("expected endDate at the end of 2026-08-03 but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
}

func TestTimelogListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogList.String(), map[string]any{
		"project_id":           float64(123),
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	})
}

func TestTimelogListByTask(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogList.String(), map[string]any{
		"task_id":              float64(123),
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	})
}
