//nolint:lll
package twprojects_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
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
		"status":      "ACTIVE",
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

// TestProjectBudgetListStatusFilter pins the status filter onto the outgoing
// query. The tool used to advertise "complete", which the budgets endpoint
// rejects with 400 "unknown project budget status".
func TestProjectBudgetListStatusFilter(t *testing.T) {
	for _, status := range []string{"UPCOMING", "ACTIVE", "COMPLETED"} {
		t.Run(status, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(`{"meta":{"page":{"hasMore":false}},"budgets":[]}`))

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
				"status": status,
			})

			if got := lastURL.Query().Get("status"); got != status {
				t.Errorf("expected status=%q in request query but got %q (raw query: %s)",
					status, got, lastURL.RawQuery)
			}
		})
	}
}

// TestProjectBudgetListStatusEnumMatchesAPI pins the published enum to the
// upstream vocabulary. The values are spelled out rather than read back from
// projectBudgetStatuses, so widening that list to something the API rejects
// fails here instead of at call time.
func TestProjectBudgetListStatusEnumMatchesAPI(t *testing.T) {
	// Accepted by GET /projects/api/v3/projects/budgets.json. Its enum also has
	// ALL, ARCHIVED, DELETED and INVALID, which this tool doesn't expose.
	want := []any{"UPCOMING", "ACTIVE", "COMPLETED"}

	schema := projectBudgetListInputSchema(t)
	property, ok := schema.Properties["status"]
	if !ok {
		t.Fatal("status property missing from input schema")
	}

	var got []any
	for _, branch := range property.AnyOf {
		if branch.Enum != nil {
			got = branch.Enum
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("status enum is %v, want %v", got, want)
	}
}

// projectBudgetListInputSchema returns the input schema
// twprojects-list_project_budgets publishes.
func projectBudgetListInputSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	group := twprojects.DefaultToolsetGroup(false, true, testutil.ProjectsEngineMock(http.StatusOK, nil))
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			if tool.Tool.Name != twprojects.MethodProjectBudgetList.String() {
				continue
			}
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				t.Fatalf("InputSchema is not *jsonschema.Schema (got %T)", tool.Tool.InputSchema)
			}
			return schema
		}
	}
	t.Fatalf("tool %s not found", twprojects.MethodProjectBudgetList)
	return nil
}

// sparseFieldsParams returns the fields[...] selections on a request query,
// keyed by entity. Tests here assert on the selected attributes, not on the
// entity key: which key a slot maps to is the SDK's contract and is covered by
// its own generated tests, so pinning the literal key would duplicate upstream
// and break whenever upstream legitimately corrects a mapping.
func sparseFieldsParams(query url.Values) map[string]string {
	selections := make(map[string]string)
	for key, values := range query {
		if entity, ok := strings.CutPrefix(key, "fields["); ok {
			if entity, ok := strings.CutSuffix(entity, "]"); ok {
				selections[entity] = strings.Join(values, ",")
			}
		}
	}
	return selections
}

// TestProjectBudgetListVerboseFalseSendsSparseFields pins the sparse fieldset
// onto the outgoing query. The endpoint implements sparse fieldsets, so
// verbose=false is honoured server-side and the handler streams the body
// untouched; asserting on the response alone could not tell a parameter that is
// sent from one that is dropped, since the mock replies with the same canned
// body either way.
//
// What is ours to guarantee is the wiring: that verbose=false populates the
// slot at all, and with the intended attributes. An omitted or empty selection
// means the API returns every field and verbose=false is a silent no-op.
func TestProjectBudgetListVerboseFalseSendsSparseFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"meta":{"page":{"hasMore":false}},"budgets":[{"id":13579,"projectId":2468}],"included":{}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"verbose": false,
	})

	selections := sparseFieldsParams(lastURL.Query())
	if len(selections) != 1 {
		t.Fatalf("expected exactly one fields[...] selection but got %v (raw query: %s)",
			selections, lastURL.RawQuery)
	}

	want := "id,projectId,type,status,capacity,capacityUsed,startDateTime,endDateTime"
	for entity, got := range selections {
		if got != want {
			t.Errorf("expected fields[%s]=%q in request query but got %q (raw query: %s)",
				entity, want, got, lastURL.RawQuery)
		}
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

	if selections := sparseFieldsParams(lastURL.Query()); len(selections) != 0 {
		t.Errorf("verbose=true must not restrict the fieldset but sent %v", selections)
	}
}
