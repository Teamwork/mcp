package twprojects_test

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// fieldsToolCase describes a list tool exposing the `fields` parameter.
//
// attributes returns the attribute names the tool's entity accepts, derived
// from the same entity struct the tool wires its sparse-fieldset slot to. A row
// asking for every one of them therefore fails loudly if a tool is wired to the
// wrong entity — the likeliest mistake in a change that touches every list tool
// — because a name valid for the intended entity is rejected by the other.
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
}}

func attributesOf[F ~string, E any]() []string {
	fields := helpers.SparseFieldNames[F, E]()
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = string(field)
	}
	return names
}

// TestListToolsExposeFields guards that every list tool advertising a `fields`
// parameter actually declares one, so a tool added later without it is caught
// rather than silently forcing callers back to the verbose switch.
func TestListToolsExposeFields(t *testing.T) {
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

// TestListToolsSendSelectedFields pins the caller's selection onto the outgoing
// query. The mock replies with the same canned body whatever is asked for, so
// asserting on the response could not distinguish a selection that is sent from
// one that is dropped — and a dropped selection means the API returns every
// attribute while the caller believes it asked for a few.
func TestListToolsSendSelectedFields(t *testing.T) {
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

// TestListToolsAlwaysSelectID guards the id the handler appends to any
// selection. Without it a caller asking only for names gets rows it cannot
// address in a follow-up get_* call, and WebLinker has no id to build a web
// link from, so it silently drops the link.
func TestListToolsAlwaysSelectID(t *testing.T) {
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

// TestListToolsFieldsOverrideVerbose covers the tools whose verbose=true branch
// sideloads related entities. A selection exists to shrink the response, so
// honouring it while still asking for the sideloads would hand back most of the
// bulk it was meant to avoid — and the sideloaded entities are not what the
// selection names.
func TestListToolsFieldsOverrideVerbose(t *testing.T) {
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

// TestListToolsVerboseFalseKeepsSelection guards the other order of arguments:
// verbose=false must not overwrite an explicit selection with its own minimal
// fieldset, which would silently return something narrower than what was asked
// for.
func TestListToolsVerboseFalseKeepsSelection(t *testing.T) {
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

// TestListToolsRejectUnknownField guards the validation that turns a guessed
// attribute name into a correctable error. The API ignores attributes it does
// not recognise, so passing one through would come back as a response quietly
// missing a field the caller asked for.
func TestListToolsRejectUnknownField(t *testing.T) {
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
					if !strings.Contains(text.Text, "must be one of") {
						t.Errorf("expected the valid attributes in the error but got %q", text.Text)
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
