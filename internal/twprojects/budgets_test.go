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

// TestProjectBudgetListVerboseFalseProjectsFields covers the case the budgets
// endpoint does not: it ignores fields[budgets], so verbose=false has to be
// enforced on the response body or it is a no-op.
func TestProjectBudgetListVerboseFalseProjectsFields(t *testing.T) {
	// A budget as the API actually returns it, ignoring the sparse fieldset.
	response := []byte(`{
		"meta":{"page":{"pageOffset":0,"pageSize":1,"count":1,"hasMore":false}},
		"budgets":[{
			"id":281044,"projectId":2468,"type":"FINANCIAL","status":"ACTIVE",
			"capacity":100000,"capacityUsed":40417,
			"startDateTime":"2026-01-01T00:00:00Z","endDateTime":"2026-12-31T00:00:00Z",
			"project":{"id":2468,"name":"Acme"},"startDate":"2026-01-01",
			"baselineCapacity":100000,"carryCapacity":0,"isRepeating":false,
			"createdByUserId":1,"createdBy":1,"dateCreated":"2026-01-01T00:00:00Z","createdAt":"2026-01-01T00:00:00Z",
			"updatedByUserId":1,"updatedBy":1,"dateUpdated":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",
			"notificationIds":[],"notifications":[],"budgetCategory":null,"financialDetailsHidden":false
		}],
		"included":{}
	}`)
	mcpServer := mcpServerMock(t, http.StatusOK, response)

	want := map[string]bool{
		"id": true, "projectId": true, "type": true, "status": true,
		"capacity": true, "capacityUsed": true, "startDateTime": true, "endDateTime": true,
	}

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"verbose": false,
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
			Meta    json.RawMessage              `json:"meta"`
		}
		if err := json.Unmarshal([]byte(textContent.Text), &payload); err != nil {
			t.Fatalf("failed to decode tool output: %v", err)
		}
		if len(payload.Budgets) != 1 {
			t.Fatalf("expected 1 budget but got %d", len(payload.Budgets))
		}
		for field := range payload.Budgets[0] {
			if !want[field] {
				t.Errorf("verbose=false leaked field %q; output should be trimmed to %v", field, keysOf(want))
			}
		}
		for field := range want {
			if _, ok := payload.Budgets[0][field]; !ok {
				t.Errorf("verbose=false dropped field %q, which is needed to join a budget to its project", field)
			}
		}
		if len(payload.Meta) == 0 {
			t.Error("verbose=false dropped meta; pagination state must survive the projection")
		}
	}))
}

// TestProjectBudgetListVerboseTrueKeepsEverything guards the other direction:
// the projection must not touch the default response.
func TestProjectBudgetListVerboseTrueKeepsEverything(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK,
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
}

func keysOf(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
