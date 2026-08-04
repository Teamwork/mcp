//nolint:lll
package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTasklistBudgetList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"meta":{"page":{"hasMore":false}},"tasklistBudgets":[{"id":98765,"projectBudgetId":12345,"tasklistId":4567}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistBudgetList.String(), map[string]any{
		"project_budget_id": float64(12345),
		"page":              float64(1),
		"page_size":         float64(10),
	})

}

func TestProjectBudgetList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"meta":{"page":{"hasMore":false}},"budgets":[{"id":13579,"projectId":2468}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"project_ids": []float64{2468, 9753},
		"status":      "active",
		"limit":       float64(5),
		"page_size":   float64(2),
		"cursor":      "next-cursor-token",
	})
}

// TestProjectBudgetListPagination pins the pagination parameters onto the
// outgoing query. The tool previously exposed no page parameter at all, so a
// caller trying to advance had nothing to set: page was silently dropped and
// every request returned page one.
func TestProjectBudgetListPagination(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"meta":{"page":{"pageOffset":1,"pageSize":250,"hasMore":true}},"budgets":[{"id":13579,"projectId":2468}]}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"page":      float64(2),
		"page_size": float64(250),
	})

	query := lastURL.Query()
	if got := query.Get("page"); got != "2" {
		t.Errorf("expected page=2 in request query but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
	if got := query.Get("pageSize"); got != "250" {
		t.Errorf("expected pageSize=250 in request query but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
}

// TestProjectBudgetListVerboseFalseSendsSparseFields pins fields[budgets] onto
// the outgoing query. The endpoint now implements sparse fieldsets under that
// key, so verbose=false is honoured server-side and the handler streams the
// body untouched; asserting on the response alone could not tell a parameter
// that is sent from one that is dropped, since the mock replies with the same
// canned body either way.
func TestProjectBudgetListVerboseFalseSendsSparseFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"meta":{"page":{"hasMore":false}},"budgets":[{"id":13579,"projectId":2468}],"included":{}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"verbose": false,
	})

	got := lastURL.Query().Get("fields[budgets]")
	want := "id,projectId,type,status,capacity,capacityUsed,startDateTime,endDateTime"
	if got != want {
		t.Errorf("expected fields[budgets]=%q in request query but got %q (raw query: %s)",
			want, got, lastURL.RawQuery)
	}
}

// TestProjectBudgetListVerboseTrueRequestsAllFields guards the other
// direction: verbose=true must not restrict the fieldset.
func TestProjectBudgetListVerboseTrueRequestsAllFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"meta":{"page":{"hasMore":false}},"budgets":[{"id":13579,"projectId":2468,"budgetCategory":"retainer"}]}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"verbose": true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		textContent, ok := toolResult.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type: %T", toolResult.Content[0])
		}

		var payload struct {
			Budgets []map[string]json.RawMessage `json:"budgets"`
		}
		if err := json.Unmarshal([]byte(textContent.Text), &payload); err != nil {
			t.Fatalf("failed to decode tool output: %v", err)
		}
		if _, ok := payload.Budgets[0]["budgetCategory"]; !ok {
			t.Error("verbose=true must pass the response through untouched")
		}
	}))

	if got := lastURL.Query().Get("fields[budgets]"); got != "" {
		t.Errorf("verbose=true must not send fields[budgets] but got %q", got)
	}
}
