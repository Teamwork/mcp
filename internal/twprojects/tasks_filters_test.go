package twprojects_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// taskFilterTools are the two tools that share the task filter surface. Every
// filter added to list_tasks has to answer a count question too, or "how many
// tasks are late" costs a full read of the rows it was going to discard.
var taskFilterTools = []string{
	twprojects.MethodTaskList.String(),
	twprojects.MethodTaskCount.String(),
}

// taskFilterTool returns the registered tool, so a handler can be driven past
// the SDK's schema validation.
func taskFilterTool(t *testing.T, method string) toolsets.ToolWrapper {
	t.Helper()

	engine := testutil.ProjectsEngineMock(http.StatusOK, []byte(`{}`))
	for _, toolset := range twprojects.DefaultToolsetGroup(false, true, engine).Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			if tool.Tool.Name == method {
				return tool
			}
		}
	}
	t.Fatalf("%s is not registered", method)
	return toolsets.ToolWrapper{}
}

// taskFilterProperty returns one published parameter of a task filter tool.
func taskFilterProperty(t *testing.T, method, name string) *jsonschema.Schema {
	t.Helper()

	schema, ok := toolInputSchemas(t)[method]
	if !ok {
		t.Fatalf("%s is not registered", method)
	}
	property, ok := schema.Properties[name]
	if !ok {
		t.Fatalf("%s does not declare %s", method, name)
	}
	return property
}

// TestTaskDateFilterReachesTheWire drives every value the tools publish for
// date_filter through the query string, reading the vocabulary off the schema
// rather than restating it: a value published without being bound would
// otherwise be accepted, dropped, and answered with the unfiltered list. The
// endpoint rejects a value it does not know with 400, so it cannot tell us
// either — and the mocks reply with the same body whatever we send.
func TestTaskDateFilterReachesTheWire(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			values := enumOf(t, taskFilterProperty(t, method, "date_filter"))
			if !slices.Contains(values, "overdue") ||
				!slices.Contains(values, "today") ||
				!slices.Contains(values, "started") {
				t.Errorf("expected overdue, today and started to be published but got %q", values)
			}

			for _, value := range values {
				t.Run(value, func(t *testing.T) {
					mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
						[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
					testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
						"project_id":  float64(777),
						"date_filter": value,
					})

					if got := sentValues(*urls, "taskFilter"); !slices.Contains(got, value) {
						t.Errorf("expected taskFilter=%q to reach the wire but sent %q (queries: %s)",
							value, got, rawQueries(*urls))
					}
				})
			}
		})
	}
}

// TestTaskDateFilterOmittedSendsNothing pins the decision not to default the
// filter. The endpoint applies "anytime" — no date restriction — when the
// parameter is absent, so filling it in here would silently narrow the results
// of every caller that never asked about dates.
func TestTaskDateFilterOmittedSendsNothing(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
				[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"project_id": float64(777),
			})

			if got := sentValues(*urls, "taskFilter"); len(got) > 0 {
				t.Errorf("expected no taskFilter but sent %q (queries: %s)", got, rawQueries(*urls))
			}
		})
	}
}

// TestTaskDateFilterRejectsAnUnknownValue covers the handler half of the
// vocabulary. The SDK validates the input schema and not the handler, so a
// client that skips validation would otherwise reach the endpoint with a value
// it answers 400 to — which is why the handler is driven directly here.
func TestTaskDateFilterRejectsAnUnknownValue(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			tool := taskFilterTool(t, method)
			result, err := tool.Handler(t.Context(), &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name:      method,
					Arguments: []byte(`{"project_id":777,"date_filter":"whenever"}`),
				},
			})
			if err != nil {
				t.Fatalf("expected a tool result rather than a Go error: %v", err)
			}
			if !result.IsError {
				t.Errorf("%s accepted an unknown date_filter value", method)
			}
		})
	}
}

// TestTaskOnlyCompletedReachesTheWire pins the completed-work filter, which is
// what replaced the endpoint's `completed` date-filter value: that value stood
// in place of a date filter, while this flag combines with one. The pairing the
// description warns about is pinned too — overdue never matches a completed
// task, so the endpoint answers the combination with an empty list rather than
// an error, and only the query string shows both filters were sent.
func TestTaskOnlyCompletedReachesTheWire(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
				[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"project_id":     float64(777),
				"only_completed": true,
				"date_filter":    "overdue",
			})

			if got := sentValues(*urls, "completedOnly"); !slices.Contains(got, "true") {
				t.Errorf("expected completedOnly=true to reach the wire but sent %q (queries: %s)",
					got, rawQueries(*urls))
			}
			if got := sentValues(*urls, "taskFilter"); !slices.Contains(got, "overdue") {
				t.Errorf("expected the date filter to survive alongside only_completed but sent %q "+
					"(queries: %s)", got, rawQueries(*urls))
			}
		})
	}
}

// TestTaskDateFilterPublishesNoCompletedWorkValues keeps the vocabulary to
// statements about dates. The endpoint also accepts `all` and `completed`
// there, and both are traps: each replaces the date filter rather than
// combining with it, and each overrides show_completed. show_completed and
// only_completed cover the same ground compositionally.
func TestTaskDateFilterPublishesNoCompletedWorkValues(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			for _, value := range enumOf(t, taskFilterProperty(t, method, "date_filter")) {
				if value == "all" || value == "completed" {
					t.Errorf("date_filter publishes %q, which switches completed work rather than "+
						"filtering by date", value)
				}
			}
		})
	}
}

// TestTaskStartAfterReachesTheWire pins the start-date bound, and that nothing
// sends the endpoint's companion endDate: with both set the endpoint stops
// reading startDate as a start-date bound and switches to a due-date window,
// which no parameter here asks for.
func TestTaskStartAfterReachesTheWire(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
				[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"project_id":  float64(777),
				"start_after": "2026-03-04",
				"due_before":  "2026-05-06",
			})

			if got := sentValues(*urls, "startDate"); !slices.Contains(got, "2026-03-04") {
				t.Errorf("expected startDate=%q to reach the wire but sent %q (queries: %s)",
					"2026-03-04", got, rawQueries(*urls))
			}
			if got := sentValues(*urls, "endDate"); len(got) > 0 {
				t.Errorf("expected no endDate but sent %q (queries: %s)", got, rawQueries(*urls))
			}
			if got := sentValues(*urls, "dueBefore"); !slices.Contains(got, "2026-05-06") {
				t.Errorf("expected dueBefore to survive alongside start_after but sent %q (queries: %s)",
					got, rawQueries(*urls))
			}
		})
	}
}

// TestTaskExcludeAssigneesReachesTheWire pins the exclusion filter, alongside
// the assignee filter it narrows. Both map to parameter names that share a
// prefix, and the mocks answer the same body either way, so only the query
// string shows whether each one travelled.
func TestTaskExcludeAssigneesReachesTheWire(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
				[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"project_id":                float64(777),
				"assignee_user_ids":         []any{float64(12345)},
				"exclude_assignee_user_ids": []any{float64(777), float64(888)},
			})

			if got := sentValues(*urls, "excludeResponsiblePartyIds"); !slices.Contains(got, "777,888") {
				t.Errorf("expected excludeResponsiblePartyIds=%q to reach the wire but sent %q (queries: %s)",
					"777,888", got, rawQueries(*urls))
			}
			if got := sentValues(*urls, "responsiblePartyIds"); !slices.Contains(got, "12345") {
				t.Errorf("expected responsiblePartyIds to survive alongside the exclusion but sent %q "+
					"(queries: %s)", got, rawQueries(*urls))
			}
		})
	}
}

// TestTaskExcludeAssigneesOmittedSendsNothing keeps an unasked-for exclusion off
// the wire: an empty list rendered as excludeResponsiblePartyIds= would be a
// parameter the endpoint has to interpret.
func TestTaskExcludeAssigneesOmittedSendsNothing(t *testing.T) {
	for _, method := range taskFilterTools {
		t.Run(method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK,
				[]byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"project_id": float64(777),
			})

			if got := sentValues(*urls, "excludeResponsiblePartyIds"); len(got) > 0 {
				t.Errorf("expected no excludeResponsiblePartyIds but sent %q (queries: %s)",
					got, rawQueries(*urls))
			}
		})
	}
}
