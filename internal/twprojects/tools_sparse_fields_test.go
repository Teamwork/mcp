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
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// fieldsToolCase describes a tool exposing the `fields` parameter.
//
// attributes returns the attribute names the tool's entity accepts, derived
// from the same entity struct the tool wires its sparse-fieldset slot to. A row
// asking for every one of them therefore fails loudly if a tool is wired to the
// wrong entity — the likeliest mistake in a change that touches every list and
// get tool — because a name valid for the intended entity is rejected by the
// other.
type fieldsToolCase struct {
	method     string
	args       map[string]any
	attributes func() []string
}

var fieldsToolCases = []fieldsToolCase{{
	method:     twprojects.MethodActivityList.String(),
	attributes: attributesOf[projects.ActivityField, projects.Activity],
}, {
	method:     twprojects.MethodCalendarList.String(),
	attributes: attributesOf[projects.CalendarField, projects.Calendar],
}, {
	method:     twprojects.MethodCalendarEventList.String(),
	args:       map[string]any{"calendar_id": float64(123)},
	attributes: attributesOf[projects.CalendarEventField, projects.CalendarEvent],
}, {
	method:     twprojects.MethodCommentList.String(),
	attributes: attributesOf[projects.CommentField, projects.Comment],
}, {
	method:     twprojects.MethodCompanyList.String(),
	attributes: attributesOf[projects.CompanyField, projects.Company],
}, {
	method:     twprojects.MethodCustomFieldList.String(),
	attributes: attributesOf[projects.CustomFieldField, projects.CustomField],
}, {
	method:     twprojects.MethodCustomFieldValueList.String(),
	args:       map[string]any{"entity": "task", "entity_id": float64(123)},
	attributes: attributesOf[projects.CustomFieldValueField, projects.CustomFieldValue],
}, {
	method:     twprojects.MethodJobRoleList.String(),
	attributes: attributesOf[projects.JobRoleField, projects.JobRole],
}, {
	method:     twprojects.MethodLinkList.String(),
	attributes: attributesOf[projects.LinkField, projects.Link],
}, {
	method:     twprojects.MethodMessageList.String(),
	attributes: attributesOf[projects.MessageField, projects.Message],
}, {
	method:     twprojects.MethodMessageReplyList.String(),
	attributes: attributesOf[projects.MessageReplyField, projects.MessageReply],
}, {
	method:     twprojects.MethodMilestoneList.String(),
	attributes: attributesOf[projects.MilestoneField, projects.Milestone],
}, {
	method:     twprojects.MethodNotebookList.String(),
	attributes: attributesOf[projects.NotebookField, projects.Notebook],
}, {
	method:     twprojects.MethodProjectBudgetList.String(),
	attributes: attributesOf[projects.ProjectBudgetField, projects.ProjectBudget],
}, {
	method:     twprojects.MethodProjectCategoryList.String(),
	attributes: attributesOf[projects.ProjectCategoryField, projects.ProjectCategory],
}, {
	method:     twprojects.MethodProjectList.String(),
	attributes: attributesOf[projects.ProjectField, projects.Project],
}, {
	method:     twprojects.MethodSkillList.String(),
	attributes: attributesOf[projects.SkillField, projects.Skill],
}, {
	method:     twprojects.MethodTagList.String(),
	attributes: attributesOf[projects.TagField, projects.Tag],
}, {
	method:     twprojects.MethodTaskList.String(),
	attributes: attributesOf[projects.TaskField, projects.Task],
}, {
	method:     twprojects.MethodTasklistList.String(),
	attributes: attributesOf[projects.TasklistField, projects.Tasklist],
}, {
	method:     twprojects.MethodTasklistBudgetList.String(),
	args:       map[string]any{"project_budget_id": float64(123)},
	attributes: attributesOf[projects.TasklistBudgetField, projects.TasklistBudget],
}, {
	method:     twprojects.MethodTeamList.String(),
	attributes: attributesOf[projects.TeamField, projects.Team],
}, {
	method:     twprojects.MethodTimelogList.String(),
	attributes: attributesOf[projects.TimelogField, projects.Timelog],
}, {
	method:     twprojects.MethodTimerList.String(),
	attributes: attributesOf[projects.TimerField, projects.Timer],
}, {
	method:     twprojects.MethodUserList.String(),
	attributes: attributesOf[projects.UserField, projects.User],
}, {
	method:     twprojects.MethodWorkflowList.String(),
	attributes: attributesOf[projects.WorkflowField, projects.Workflow],
}, {
	method:     twprojects.MethodWorkflowStageList.String(),
	args:       map[string]any{"workflow_id": float64(123)},
	attributes: attributesOf[projects.WorkflowStageField, projects.WorkflowStage],
}, {
	method:     twprojects.MethodCommentGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.CommentField, projects.Comment],
}, {
	method:     twprojects.MethodCompanyGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.CompanyField, projects.Company],
}, {
	method:     twprojects.MethodJobRoleGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.JobRoleField, projects.JobRole],
}, {
	method:     twprojects.MethodMessageGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.MessageField, projects.Message],
}, {
	method:     twprojects.MethodMessageReplyGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.MessageReplyField, projects.MessageReply],
}, {
	method:     twprojects.MethodMilestoneGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.MilestoneField, projects.Milestone],
}, {
	method:     twprojects.MethodNotebookGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.NotebookField, projects.Notebook],
}, {
	method:     twprojects.MethodProjectCategoryGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.ProjectCategoryField, projects.ProjectCategory],
}, {
	method:     twprojects.MethodProjectGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.ProjectField, projects.Project],
}, {
	method:     twprojects.MethodTaskGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.TaskField, projects.Task],
}, {
	method:     twprojects.MethodTasklistGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.TasklistField, projects.Tasklist],
}, {
	method:     twprojects.MethodTimelogGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.TimelogField, projects.Timelog],
}, {
	method:     twprojects.MethodTimerGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.TimerField, projects.Timer],
}, {
	method:     twprojects.MethodUserGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.UserField, projects.User],
}, {
	method:     twprojects.MethodWorkflowGet.String(),
	args:       map[string]any{"id": float64(123)},
	attributes: attributesOf[projects.WorkflowField, projects.Workflow],
}, {
	method:     twprojects.MethodWorkflowStageGet.String(),
	args:       map[string]any{"workflow_id": float64(123), "id": float64(456)},
	attributes: attributesOf[projects.WorkflowStageField, projects.WorkflowStage],
}}

func attributesOf[F ~string, E any]() []string {
	fields := helpers.SparseFieldNames[F, E]()
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = string(field)
	}
	return names
}

// TestSparseFieldsToolsAreCovered ties the table above to the tools that
// actually declare a `fields` parameter, in both directions: a tool wired up
// without a row here is untested, and a row naming a tool that dropped the
// parameter is stale.
func TestSparseFieldsToolsAreCovered(t *testing.T) {
	engine := testutil.ProjectsEngineMock(http.StatusOK, []byte(`{}`))
	group := twprojects.DefaultToolsetGroup(false, true, engine)

	declared := make(map[string]bool)
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				continue
			}
			if _, ok := schema.Properties["fields"]; ok {
				declared[tool.Tool.Name] = true
			}
		}
	}

	for _, testCase := range fieldsToolCases {
		if !declared[testCase.method] {
			t.Errorf("%s does not declare a fields parameter", testCase.method)
		}
		delete(declared, testCase.method)
	}
	for name := range declared {
		t.Errorf("%s declares a fields parameter but is not covered by fieldsToolCases", name)
	}
}

// TestSparseFieldsSchemaEnumMatchesValidator pins the vocabulary a client sees
// before calling to the one the tool accepts. An enum that is missing, short a
// name, or reflected off the wrong entity puts the model back to learning the
// names one rejection at a time. Riding the table TestSparseFieldsToolsAreCovered
// keeps in step with the toolset covers future tools automatically.
func TestSparseFieldsSchemaEnumMatchesValidator(t *testing.T) {
	engine := testutil.ProjectsEngineMock(http.StatusOK, []byte(`{}`))
	group := twprojects.DefaultToolsetGroup(false, true, engine)

	schemas := make(map[string]*jsonschema.Schema)
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				continue
			}
			if fields, ok := schema.Properties["fields"]; ok {
				schemas[tool.Tool.Name] = fields
			}
		}
	}

	for _, testCase := range fieldsToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			schema, ok := schemas[testCase.method]
			if !ok {
				t.Fatalf("%s does not declare a fields parameter", testCase.method)
			}
			enum := fieldsSchemaEnum(t, schema)
			if want := testCase.attributes(); !slices.Equal(enum, want) {
				t.Errorf("expected the fields enum to be %v but got %v", want, enum)
			}
		})
	}
}

// fieldsSchemaEnum reads the names enumerated by a `fields` schema, from the
// array branch's item schema where a client looks for them.
func fieldsSchemaEnum(t *testing.T, schema *jsonschema.Schema) []string {
	t.Helper()

	for _, branch := range schema.AnyOf {
		if branch.Type != "array" {
			continue
		}
		if branch.Items == nil {
			t.Fatal("expected the array branch to declare items")
		}
		names := make([]string, 0, len(branch.Items.Enum))
		for _, value := range branch.Items.Enum {
			name, ok := value.(string)
			if !ok {
				t.Fatalf("expected string enum values but got %T", value)
			}
			names = append(names, name)
		}
		return names
	}
	t.Fatal("expected an array branch in the fields schema")
	return nil
}

// TestSparseFieldsSendSelection pins the caller's selection onto the outgoing
// query. The mock replies with the same canned body whatever is asked for, so
// asserting on the response could not distinguish a selection that is sent from
// one that is dropped — and a dropped selection means the API returns every
// attribute while the caller believes it asked for a few.
func TestSparseFieldsSendSelection(t *testing.T) {
	for _, testCase := range fieldsToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))

			attributes := testCase.attributes()
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, argsWithFields(testCase.args, attributes...))

			selections := sparseFieldsParams(lastURL.Query())
			if len(selections) != 1 {
				t.Fatalf("expected exactly one fields[...] selection but got %v (raw query: %s)",
					selections, lastURL.RawQuery)
			}
			want := strings.Join(attributes, ",")
			for entity, got := range selections {
				if got != want {
					t.Errorf("expected fields[%s]=%q in request query but got %q", entity, want, got)
				}
			}
		})
	}
}

// TestSparseFieldsAlwaysSelectID guards the id the handler appends to any
// selection. Without it a caller asking only for names gets rows it cannot
// address in a follow-up get_* call, and WebLinker has no id to build a web
// link from, so it silently drops the link.
func TestSparseFieldsAlwaysSelectID(t *testing.T) {
	for _, testCase := range fieldsToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			attributes := testCase.attributes()
			var selection string
			for _, attribute := range attributes {
				if attribute != "id" {
					selection = attribute
					break
				}
			}
			if selection == "" {
				t.Skipf("%s has no attribute other than id", testCase.method)
			}

			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, argsWithFields(testCase.args, selection))

			for entity, got := range sparseFieldsParams(lastURL.Query()) {
				if !slices.Contains(strings.Split(got, ","), "id") {
					t.Errorf("expected id in fields[%s] but got %q", entity, got)
				}
			}
		})
	}
}

// TestSparseFieldsGetOmitsUnselectedAttributes is the whole reason a selection
// on a get_* tool streams the API body instead of re-marshalling the SDK's
// typed response: those entity structs carry no `omitempty`, so every attribute
// the caller left out would come back as a zero value that reads like real data
// — a task with `"dueDate": null` is indistinguishable from a task with no due
// date. The control below pins that difference.
func TestSparseFieldsGetOmitsUnselectedAttributes(t *testing.T) {
	const response = `{"task":{"id":7,"name":"Ship it"}}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(response))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(), map[string]any{
		"id":     float64(7),
		"fields": []any{"name"},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		got := slices.Sorted(maps.Keys(taskFromToolResult(t, result)))
		if want := []string{"id", "name"}; !slices.Equal(got, want) {
			t.Errorf("expected only the selected attributes %v but got %v", want, got)
		}
	}))

	// Control: without a selection the tool keeps re-marshalling the typed
	// response, which reports every attribute the SDK models whether the API
	// returned it or not. That is the existing behaviour, left untouched.
	mcpServer = mcpServerMock(t, http.StatusOK, []byte(response))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskGet.String(), map[string]any{
		"id": float64(7),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		if _, ok := taskFromToolResult(t, result)["dueDate"]; !ok {
			t.Error("expected the unselected path to report every modelled attribute")
		}
	}))
}

// TestSparseFieldsGetDropsSideloads guards that a selection on a get_* tool
// stops asking for the sideloads the handler requests by default: they are not
// what the selection named, and they carry more bulk than the attributes it
// excluded.
func TestSparseFieldsGetDropsSideloads(t *testing.T) {
	for _, method := range []string{
		twprojects.MethodTaskGet.String(),
		twprojects.MethodProjectGet.String(),
		twprojects.MethodCompanyGet.String(),
	} {
		t.Run(method, func(t *testing.T) {
			// Control: the tool does sideload without a selection, so the assertion
			// below fails for the right reason.
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{"id": float64(7)})
			if lastURL.Query().Get("include") == "" {
				t.Fatalf("expected the tool to sideload by default (raw query: %s)", lastURL.RawQuery)
			}

			mcpServer, lastURL = testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"id":     float64(7),
				"fields": []any{"name"},
			})
			if got := lastURL.Query().Get("include"); got != "" {
				t.Errorf("expected no sideloads alongside a field selection but got include=%q", got)
			}
		})
	}
}

// taskFromToolResult decodes the task object out of a get_task tool result.
func taskFromToolResult(t *testing.T, result mcp.Result) map[string]json.RawMessage {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	var payload struct {
		Task map[string]json.RawMessage `json:"task"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	return payload.Task
}

// TestSparseFieldsOverrideVerbose covers the list tools whose verbose=true branch
// sideloads related entities. A selection exists to shrink the response, so
// honouring it while still asking for the sideloads would hand back most of the
// bulk it was meant to avoid — and the sideloaded entities are not what the
// selection names.
func TestSparseFieldsOverrideVerbose(t *testing.T) {
	for _, method := range []string{
		twprojects.MethodTaskList.String(),
		twprojects.MethodProjectList.String(),
		twprojects.MethodCompanyList.String(),
	} {
		t.Run(method, func(t *testing.T) {
			// Control: verbose=true on its own does sideload, so the assertion
			// below fails for the right reason rather than because the tool never
			// sideloads at all.
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{"verbose": true})
			if lastURL.Query().Get("include") == "" {
				t.Fatalf("expected verbose=true to sideload (raw query: %s)", lastURL.RawQuery)
			}

			mcpServer, lastURL = testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, method, map[string]any{
				"verbose": true,
				"fields":  []any{"name"},
			})

			query := lastURL.Query()
			if got := query.Get("include"); got != "" {
				t.Errorf("expected no sideloads alongside a field selection but got include=%q", got)
			}
			selections := sparseFieldsParams(query)
			if len(selections) != 1 {
				t.Fatalf("expected exactly one fields[...] selection but got %v (raw query: %s)",
					selections, lastURL.RawQuery)
			}
			for entity, got := range selections {
				if got != "name,id" {
					t.Errorf("expected fields[%s]=%q in request query but got %q", entity, "name,id", got)
				}
			}
		})
	}
}

// TestSparseFieldsVerboseFalseKeepsSelection guards the other order of arguments:
// verbose=false must not overwrite an explicit selection with its own minimal
// fieldset, which would silently return something narrower than what was asked
// for.
func TestSparseFieldsVerboseFalseKeepsSelection(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"verbose": false,
		"fields":  []any{"dueDate"},
	})

	for entity, got := range sparseFieldsParams(lastURL.Query()) {
		if got != "dueDate,id" {
			t.Errorf("expected fields[%s]=%q in request query but got %q", entity, "dueDate,id", got)
		}
	}
}

// TestSparseFieldsPredecessorsCarryRelatedTasks guards the one attribute a
// selection cannot obtain on its own. The API leaves `predecessors` empty unless
// the request also sets includeRelatedTasks, and it reports nothing about the
// omission — so selecting it without the filter returns `"predecessors": []` on
// every row, which reads exactly like a task nothing blocks. That is how a
// dependency question gets answered "nothing is blocking".
//
// It has to be asserted on the query: the mock replies with the same canned body
// whether the filter is sent or not.
func TestSparseFieldsPredecessorsCarryRelatedTasks(t *testing.T) {
	for _, testCase := range []struct {
		method string
		args   map[string]any
	}{
		{method: twprojects.MethodTaskList.String()},
		{method: twprojects.MethodTaskGet.String(), args: map[string]any{"id": float64(7)}},
	} {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method,
				argsWithFields(testCase.args, "predecessors"))
			if got := lastURL.Query().Get("includeRelatedTasks"); got != "true" {
				t.Errorf("expected includeRelatedTasks=true alongside a predecessors selection but got %q "+
					"(raw query: %s)", got, lastURL.RawQuery)
			}

			// Control: the filter rides on the selection naming predecessors, not on
			// every selection, so an unrelated one must not carry it.
			mcpServer, lastURL = testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method,
				argsWithFields(testCase.args, "name"))
			if got := lastURL.Query().Get("includeRelatedTasks"); got != "" {
				t.Errorf("expected no includeRelatedTasks without a predecessors selection but got %q", got)
			}
		})
	}
}

// TestSparseFieldsRejectUnknownValue guards the validation that turns a guessed
// attribute name into a correctable error. The API ignores attributes it does
// not recognise, so passing one through would come back as a response quietly
// missing a field the caller asked for.
//
// Which layer rejects it is not asserted: with the enum in the schema the SDK
// validates before the handler runs. Either way the error must name the
// rejected value and every accepted one, so recovery costs one round trip.
func TestSparseFieldsRejectUnknownValue(t *testing.T) {
	for _, testCase := range fieldsToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method,
				argsWithFields(testCase.args, "notAnAttribute"),
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok {
						t.Fatalf("unexpected result type: %T", result)
					}
					if !toolResult.IsError {
						t.Fatal("expected an unknown attribute to fail the call")
					}
					text, ok := toolResult.Content[0].(*mcp.TextContent)
					if !ok {
						t.Fatalf("unexpected content type: %T", toolResult.Content[0])
					}
					if !strings.Contains(text.Text, "notAnAttribute") {
						t.Errorf("expected the rejected attribute in the error but got %q", text.Text)
					}
					for _, attribute := range testCase.attributes() {
						if !strings.Contains(text.Text, attribute) {
							t.Errorf("expected the valid attribute %q in the error but got %q", attribute, text.Text)
						}
					}
				}))
		})
	}
}

// argsWithFields copies the tool's required arguments and adds a fields
// selection.
func argsWithFields(base map[string]any, fields ...string) map[string]any {
	args := make(map[string]any, len(base)+1)
	maps.Copy(args, base)
	selection := make([]any, len(fields))
	for i, field := range fields {
		selection[i] = field
	}
	args["fields"] = selection
	return args
}
