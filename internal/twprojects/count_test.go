package twprojects_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// countOnlyToolCases lists every tool exposing count_only, with the arguments it
// needs to reach the API. A list tool belongs here when its SDK filters carry a
// CountMode slot.
//
// Left out, because they cannot answer a count: list_calendar_events and search
// (cursor-paginated, no page object), list_teams and list_links (v1),
// list_industries (unpaginated), list_project_templates (no CountMode slot),
// users_workload (no SetRequest) and summarize_timelogs (own aggregate).
var countOnlyToolCases = []struct {
	method string
	args   map[string]any
}{
	{method: twprojects.MethodActivityList.String()},
	{method: twprojects.MethodCalendarList.String()},
	{method: twprojects.MethodCommentList.String()},
	{method: twprojects.MethodCompanyList.String()},
	{method: twprojects.MethodCustomFieldList.String()},
	{method: twprojects.MethodCustomFieldValueList.String(),
		args: map[string]any{"entity": "task", "entity_id": float64(123)}},
	{method: twprojects.MethodCustomItemList.String(), args: map[string]any{"project_id": float64(123)}},
	{method: twprojects.MethodCustomItemFieldList.String(), args: map[string]any{"custom_item_id": float64(123)}},
	{method: twprojects.MethodCustomItemRecordList.String(), args: map[string]any{"custom_item_id": float64(123)}},
	{method: twprojects.MethodJobRoleList.String()},
	{method: twprojects.MethodMessageList.String()},
	{method: twprojects.MethodMessageReplyList.String()},
	{method: twprojects.MethodMilestoneList.String()},
	{method: twprojects.MethodNotebookList.String()},
	{method: twprojects.MethodProjectBudgetList.String()},
	{method: twprojects.MethodProjectCategoryList.String()},
	{method: twprojects.MethodProjectList.String()},
	{method: twprojects.MethodSkillList.String()},
	{method: twprojects.MethodTagList.String()},
	{method: twprojects.MethodTaskList.String()},
	{method: twprojects.MethodTasklistList.String()},
	{method: twprojects.MethodTasklistBudgetList.String(), args: map[string]any{"project_budget_id": float64(123)}},
	{method: twprojects.MethodTimelogList.String()},
	{method: twprojects.MethodTimerList.String()},
	{method: twprojects.MethodUserList.String()},
	{method: twprojects.MethodWorkflowList.String()},
	{method: twprojects.MethodWorkflowStageList.String(), args: map[string]any{"workflow_id": float64(123)}},
}

// countOnlyResponse carries rows the count path must drop and the count it must
// report. Every tool shares it: the count path reads meta.page.count only.
const countOnlyResponse = `{"tasks":[{"id":1,"name":"Ship it"}],"included":{"customfields":{}},` +
	`"meta":{"page":{"count":42,"hasMore":true}}}`

// TestCountOnlyToolsAreCovered ties the table to the tools declaring count_only,
// both ways: a tool without a row is untested, a row without a tool is stale.
func TestCountOnlyToolsAreCovered(t *testing.T) {
	declared := make(map[string]bool)
	for name, schema := range toolInputSchemas(t) {
		if _, ok := schema.Properties["count_only"]; ok {
			declared[name] = true
		}
	}

	for _, testCase := range countOnlyToolCases {
		if !declared[testCase.method] {
			t.Errorf("%s does not declare a count_only parameter", testCase.method)
		}
		delete(declared, testCase.method)
	}
	for name := range declared {
		t.Errorf("%s declares a count_only parameter but is not covered by countOnlyToolCases", name)
	}
}

// TestCountOnlyAsksForAnExactCount is what every count's correctness rests on,
// and it has to be asserted on the query: the mock replies identically either
// way, while the API would answer skipCounts=true with a lower bound of 2.
func TestCountOnlyAsksForAnExactCount(t *testing.T) {
	for _, testCase := range countOnlyToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(countOnlyResponse))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, argsWithCountOnly(testCase.args))

			query := lastURL.Query()
			for parameter, want := range map[string]string{
				"skipCounts": "false",
				"page":       "1",
				"pageSize":   "1",
			} {
				if got := query.Get(parameter); got != want {
					t.Errorf("expected %s=%s in the count query but got %q (raw query: %s)",
						parameter, want, got, lastURL.RawQuery)
				}
			}
		})
	}
}

// TestCountOnlyAsksForNothingPerRow covers the other half of the rewiring: no
// rows come back, so the sideloads and fieldsets are work nobody reads.
func TestCountOnlyAsksForNothingPerRow(t *testing.T) {
	for _, testCase := range countOnlyToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(countOnlyResponse))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, argsWithCountOnly(testCase.args))

			for parameter := range lastURL.Query() {
				if parameter == "include" || strings.HasPrefix(parameter, "fields[") {
					t.Errorf("expected no per-row parameters in the count query but got %s (raw query: %s)",
						parameter, lastURL.RawQuery)
				}
			}
		})
	}
}

// TestCountOnlyDropsCursorPaging covers the one list tool that also paginates by
// cursor: an endpoint handed one ignores page and pageSize, undoing the single
// row the count rides on.
func TestCountOnlyDropsCursorPaging(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(countOnlyResponse))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"count_only": true,
		"cursor":     "opaque",
		"limit":      float64(200),
	})

	query := lastURL.Query()
	for _, parameter := range []string{"cursor", "limit"} {
		if _, ok := query[parameter]; ok {
			t.Errorf("expected no %s in the count query but got %s", parameter, lastURL.RawQuery)
		}
	}
	if got := query.Get("pageSize"); got != "1" {
		t.Errorf("expected pageSize=1 but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
}

// TestCountOnlyReturnsOnlyTheCount pins the response shape: the point of the
// parameter is the payload it removes.
func TestCountOnlyReturnsOnlyTheCount(t *testing.T) {
	for _, testCase := range countOnlyToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer := mcpServerMock(t, http.StatusOK, []byte(countOnlyResponse))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, argsWithCountOnly(testCase.args),
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					testutil.CheckMessage(t, result)

					payload := payloadFromToolResult(t, result)
					if got := slices.Sorted(maps.Keys(payload)); !slices.Equal(got, []string{"count"}) {
						t.Fatalf("expected only a count but got %v", got)
					}
					if got := string(payload["count"]); got != "42" {
						t.Errorf("expected count 42 but got %s", got)
					}
				}))
		})
	}
}

// TestCountOnlyWithoutACountIsAnError guards the number nobody should receive:
// with no count reported, a zero would read as "nothing matches".
func TestCountOnlyWithoutACountIsAnError(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"tasks":[],"meta":{"page":{"hasMore":false}}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(),
		map[string]any{"count_only": true},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()

			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if !toolResult.IsError {
				t.Fatal("expected a tool error when the API reports no count")
			}
		}))
}

// TestCountOnlyLeavesTheListPathAlone is the control for the feature: the count
// path is an extra answer, not a change to the rows.
func TestCountOnlyLeavesTheListPathAlone(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(countOnlyResponse))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"count_only": false,
		"page_size":  float64(200),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		if _, ok := payloadFromToolResult(t, result)["tasks"]; !ok {
			t.Error("expected the list path to still return rows")
		}
	}))
	if got := lastURL.Query().Get("pageSize"); got != "200" {
		t.Errorf("expected the caller's page size to survive but got %q", got)
	}
	if _, ok := lastURL.Query()["skipCounts"]; ok {
		t.Errorf("expected the list path to leave skipCounts to the API but got %s", lastURL.RawQuery)
	}
}

// countToolCases pairs every derived count tool with the list tool it is
// derived from, and the arguments both accept.
var countToolCases = []struct {
	method string
	list   string
	args   map[string]any
}{{
	method: twprojects.MethodTaskCount.String(),
	list:   twprojects.MethodTaskList.String(),
	args:   map[string]any{"project_id": float64(7), "due_before": "2026-08-14"},
}, {
	method: twprojects.MethodProjectCount.String(),
	list:   twprojects.MethodProjectList.String(),
	args:   map[string]any{"search_term": "internal"},
}, {
	method: twprojects.MethodMilestoneCount.String(),
	list:   twprojects.MethodMilestoneList.String(),
	args:   map[string]any{"project_id": float64(7)},
}, {
	method: twprojects.MethodTimelogCount.String(),
	list:   twprojects.MethodTimelogList.String(),
	args:   map[string]any{"project_id": float64(7), "start_date": "2026-08-01"},
}}

// TestCountToolsAreCovered ties the table to the registered tools, so a count
// tool derived without tests fails here instead of shipping unexercised.
func TestCountToolsAreCovered(t *testing.T) {
	derived := make(map[string]bool)
	for name := range toolInputSchemas(t) {
		if strings.HasPrefix(name, "twprojects-count_") {
			derived[name] = true
		}
	}

	for _, testCase := range countToolCases {
		if !derived[testCase.method] {
			t.Errorf("%s is not registered", testCase.method)
		}
		delete(derived, testCase.method)
	}
	for name := range derived {
		t.Errorf("%s is registered but is not covered by countToolCases", name)
	}
}

// TestCountToolsReportTheSameCountAsTheirListTool pins the two paths to one
// answer, body included: a caller getting different results from each has no way
// to tell which is wrong.
func TestCountToolsReportTheSameCountAsTheirListTool(t *testing.T) {
	for _, testCase := range countToolCases {
		for method, arguments := range map[string]map[string]any{
			testCase.list:   argsWithCountOnly(testCase.args),
			testCase.method: testCase.args,
		} {
			t.Run(method, func(t *testing.T) {
				mcpServer := mcpServerMock(t, http.StatusOK, []byte(countOnlyResponse))
				testutil.ExecuteToolRequest(t, mcpServer, method, arguments,
					testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
						t.Helper()
						testutil.CheckMessage(t, result)

						payload := payloadFromToolResult(t, result)
						if got := slices.Sorted(maps.Keys(payload)); !slices.Equal(got, []string{"count"}) {
							t.Fatalf("expected only a count but got %v", got)
						}
						if got := string(payload["count"]); got != "42" {
							t.Errorf("expected count 42 but got %s", got)
						}
					}))
			})
		}
	}
}

// TestTaskCountSendsTheFiltersItWasGiven covers the count tool's delegation:
// its filters are the list tool's filters, so they have to reach the wire the
// same way, alongside the count rewiring.
func TestTaskCountSendsTheFiltersItWasGiven(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(countOnlyResponse))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCount.String(), map[string]any{
		"project_id":        float64(7),
		"assignee_user_ids": []any{float64(11), float64(12)},
		"due_before":        "2026-08-14",
		"show_completed":    true,
	})

	if got, want := lastURL.Path, "/projects/api/v3/projects/7/tasks.json"; got != want {
		t.Errorf("expected the project scope in the path %q but got %q", want, got)
	}
	query := lastURL.Query()
	for parameter, want := range map[string]string{
		"responsiblePartyIds":   "11,12",
		"dueBefore":             "2026-08-14",
		"includeCompletedTasks": "true",
		"showCompletedLists":    "true",
		"skipCounts":            "false",
		"pageSize":              "1",
	} {
		if got := query.Get(parameter); got != want {
			t.Errorf("expected %s=%s but got %q (raw query: %s)", parameter, want, got, lastURL.RawQuery)
		}
	}
}

// TestCountToolsDropRowParameters covers each derived schema: the filters carry
// over — losing one silently changes what is counted — and the row parameters
// are gone.
func TestCountToolsDropRowParameters(t *testing.T) {
	schemas := toolInputSchemas(t)

	// Restated rather than read from the package: these tests live outside it.
	// Keep in sync with countToolDroppedParams — ordering is row shaping too, so
	// its four parameters are dropped as well.
	dropped := []string{
		"page", "page_size", "cursor", "limit", "verbose", "fields", "count_only",
		"order_by", "order_mode", "order_by_custom_field_id", "order_by_field_id",
	}

	for _, testCase := range countToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			count, ok := schemas[testCase.method]
			if !ok {
				t.Fatalf("%s is not registered", testCase.method)
			}
			list, ok := schemas[testCase.list]
			if !ok {
				t.Fatalf("%s is not registered", testCase.list)
			}

			for _, name := range dropped {
				if _, ok := count.Properties[name]; ok {
					t.Errorf("expected %s not to advertise %s", testCase.method, name)
				}
			}
			for name := range list.Properties {
				if slices.Contains(dropped, name) {
					continue
				}
				if _, ok := count.Properties[name]; !ok {
					t.Errorf("expected %s to carry the %s filter of %s", testCase.method, name, testCase.list)
				}
			}
		})
	}
}

// TestCountToolsAPIFailuresAreToolResults keeps the delegation from swallowing a
// status, or turning it into a protocol-level error.
func TestCountToolsAPIFailuresAreToolResults(t *testing.T) {
	for _, testCase := range countToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			for _, status := range []int{http.StatusForbidden, http.StatusNotFound} {
				mcpServer := mcpServerMock(t, status, []byte(`{}`))
				testutil.ExecuteToolRequest(t, mcpServer, testCase.method, testCase.args,
					testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
						t.Helper()

						toolResult, ok := result.(*mcp.CallToolResult)
						if !ok {
							t.Fatalf("unexpected result type: %T", result)
						}
						if !toolResult.IsError {
							t.Errorf("status %d: expected a tool error", status)
						}
					}))
			}
		})
	}
}

// TestCountToolsAskForAnExactCount is the derived half of
// TestCountOnlyAsksForAnExactCount: the rewiring has to survive the delegation.
func TestCountToolsAskForAnExactCount(t *testing.T) {
	for _, testCase := range countToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(countOnlyResponse))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, testCase.args)

			query := lastURL.Query()
			for parameter, want := range map[string]string{
				"skipCounts": "false",
				"page":       "1",
				"pageSize":   "1",
			} {
				if got := query.Get(parameter); got != want {
					t.Errorf("expected %s=%s in the count query but got %q (raw query: %s)",
						parameter, want, got, lastURL.RawQuery)
				}
			}
		})
	}
}

// argsWithCountOnly copies args with count_only set, leaving the table's maps
// untouched between subtests.
func argsWithCountOnly(args map[string]any) map[string]any {
	withCount := make(map[string]any, len(args)+1)
	maps.Copy(withCount, args)
	withCount["count_only"] = true
	return withCount
}

// toolInputSchemas collects the input schema of every registered tool.
func toolInputSchemas(t *testing.T) map[string]*jsonschema.Schema {
	t.Helper()

	engine := testutil.ProjectsEngineMock(http.StatusOK, []byte(`{}`))
	group := twprojects.DefaultToolsetGroup(false, true, engine)

	schemas := make(map[string]*jsonschema.Schema)
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			if schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema); ok {
				schemas[tool.Tool.Name] = schema
			}
		}
	}
	return schemas
}

// payloadFromToolResult decodes the top-level object of a tool result.
func payloadFromToolResult(t *testing.T, result mcp.Result) map[string]json.RawMessage {
	t.Helper()

	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(toolResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	return payload
}

// toolResultText reads the text content of a tool result.
func toolResultText(t *testing.T, result mcp.Result) string {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if len(toolResult.Content) != 1 {
		t.Fatalf("expected exactly one content block but got %d", len(toolResult.Content))
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	return text.Text
}
