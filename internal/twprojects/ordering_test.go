package twprojects_test

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// orderingToolCase describes a tool exposing the ordering parameters.
//
// args carries whatever the tool requires besides the ordering itself, so a row
// exercises ordering and nothing else.
type orderingToolCase struct {
	method string
	args   map[string]any

	// orderModeOnly marks the endpoints that sort by a fixed column and accept a
	// direction but no order_by field.
	orderModeOnly bool

	// byParam and modeParam are the query-string parameters the endpoint reads,
	// defaulting to the v3 orderBy/orderMode. The team routes are v1 and take
	// sortBy/sortOrder instead; the SDK renames them in TeamListRequestFilters'
	// apply, so a tool wired the same way as every other still has to be
	// asserted under the other names.
	byParam   string
	modeParam string

	// sdkDefaultBy and sdkDefaultMode are what the endpoint receives when the
	// caller asks for no ordering, for the requests whose SDK constructor
	// pre-fills the filters (NewTasklistBudgetListRequest sets dateCreated/asc).
	// Empty means the omission must reach the wire as no parameter at all.
	sdkDefaultBy   string
	sdkDefaultMode string
}

func (c orderingToolCase) orderByParam() string {
	if c.byParam != "" {
		return c.byParam
	}
	return "orderBy"
}

func (c orderingToolCase) orderModeParam() string {
	if c.modeParam != "" {
		return c.modeParam
	}
	return "orderMode"
}

var orderingToolCases = []orderingToolCase{{
	method: twprojects.MethodActivityList.String(),
}, {
	method: twprojects.MethodCalendarList.String(),
}, {
	method: twprojects.MethodCalendarEventList.String(),
	args:   map[string]any{"calendar_id": float64(123)},
}, {
	method: twprojects.MethodCommentList.String(),
}, {
	method: twprojects.MethodCompanyList.String(),
}, {
	method: twprojects.MethodCustomFieldList.String(),
}, {
	method: twprojects.MethodCustomItemList.String(),
	args:   map[string]any{"project_id": float64(123)},
}, {
	method: twprojects.MethodCustomItemRecordList.String(),
	args:   map[string]any{"custom_item_id": float64(123)},
}, {
	method:        twprojects.MethodCustomItemFieldList.String(),
	args:          map[string]any{"custom_item_id": float64(123)},
	orderModeOnly: true,
}, {
	method:        twprojects.MethodJobRoleList.String(),
	orderModeOnly: true,
}, {
	method: twprojects.MethodMessageList.String(),
}, {
	method: twprojects.MethodMessageReplyList.String(),
}, {
	method: twprojects.MethodMilestoneList.String(),
}, {
	method: twprojects.MethodNotebookList.String(),
}, {
	method: twprojects.MethodProjectList.String(),
}, {
	method:        twprojects.MethodSkillList.String(),
	orderModeOnly: true,
}, {
	method: twprojects.MethodTagList.String(),
}, {
	method: twprojects.MethodTaskList.String(),
}, {
	method: twprojects.MethodTasklistList.String(),
}, {
	method:         twprojects.MethodTasklistBudgetList.String(),
	args:           map[string]any{"project_budget_id": float64(123)},
	sdkDefaultBy:   "dateCreated",
	sdkDefaultMode: "asc",
}, {
	method:    twprojects.MethodTeamList.String(),
	byParam:   "sortBy",
	modeParam: "sortOrder",
}, {
	method: twprojects.MethodTimelogList.String(),
}, {
	method: twprojects.MethodSummarizeTimelogs.String(),
	args:   map[string]any{"start_date": "2026-01-01", "end_date": "2026-01-31"},
}, {
	method: twprojects.MethodUserList.String(),
}, {
	method: twprojects.MethodWorkflowStageList.String(),
	args:   map[string]any{"workflow_id": float64(123)},
}}

// orderingSchemas returns every tool's ordering properties, keyed by tool name.
func orderingSchemas(t *testing.T) map[string]map[string]*jsonschema.Schema {
	t.Helper()

	engine := testutil.ProjectsEngineMock(http.StatusOK, []byte(`{}`))
	group := twprojects.DefaultToolsetGroup(false, true, engine)

	declared := make(map[string]map[string]*jsonschema.Schema)
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				continue
			}
			for _, name := range []string{"order_by", "order_mode"} {
				property, ok := schema.Properties[name]
				if !ok {
					continue
				}
				if declared[tool.Tool.Name] == nil {
					declared[tool.Tool.Name] = make(map[string]*jsonschema.Schema)
				}
				declared[tool.Tool.Name][name] = property
			}
		}
	}
	return declared
}

// enumOf pulls the string enum out of the AnyOf branch that carries it. Every
// ordering parameter is published as {AnyOf: [{string, enum}, {null}]} — the
// null branch is what lets an OpenAI strict-mode client, which must send every
// property, leave the ordering unset.
func enumOf(t *testing.T, schema *jsonschema.Schema) []string {
	t.Helper()

	for _, branch := range schema.AnyOf {
		if branch.Type != "string" || branch.Enum == nil {
			continue
		}
		values := make([]string, 0, len(branch.Enum))
		for _, value := range branch.Enum {
			text, ok := value.(string)
			if !ok {
				t.Fatalf("expected a string enum value but got %T", value)
			}
			values = append(values, text)
		}
		return values
	}
	t.Fatalf("no string enum found in schema")
	return nil
}

// sentValues collects what the tool sent for a query parameter, across every
// request it made. Tools that make a follow-up call (list_custom_item_records
// resolves the custom item's field schema after listing) would otherwise be
// asserted against the wrong request.
func sentValues(urls []url.URL, param string) []string {
	var values []string
	for _, u := range urls {
		values = append(values, u.Query()[param]...)
	}
	return values
}

func rawQueries(urls []url.URL) string {
	queries := make([]string, len(urls))
	for i, u := range urls {
		queries[i] = u.Path + "?" + u.RawQuery
	}
	return strings.Join(queries, " | ")
}

// TestOrderingToolsAreCovered ties the table above to the tools that actually
// declare ordering, in both directions: a tool wired up without a row here is
// untested, and a row naming a tool that dropped the parameters is stale.
func TestOrderingToolsAreCovered(t *testing.T) {
	declared := orderingSchemas(t)

	for _, testCase := range orderingToolCases {
		properties, ok := declared[testCase.method]
		if !ok {
			t.Errorf("%s does not declare any ordering parameter", testCase.method)
			continue
		}
		if _, ok := properties["order_mode"]; !ok {
			t.Errorf("%s does not declare order_mode", testCase.method)
		}
		_, hasOrderBy := properties["order_by"]
		if hasOrderBy == testCase.orderModeOnly {
			t.Errorf("%s: order_by declared = %t, want %t", testCase.method, hasOrderBy, !testCase.orderModeOnly)
		}
		delete(declared, testCase.method)
	}
	for name := range declared {
		t.Errorf("%s declares an ordering parameter but is not covered by orderingToolCases", name)
	}
}

// TestOrderingEveryPublishedValueReachesTheWire is the load-bearing test for
// these parameters, and it asserts on the query string because the mocks reply
// with the same canned body whether or not the ordering was forwarded — a
// dropped order_by looks exactly like a working one from the response alone.
//
// It drives every value the tool publishes rather than a representative one.
// The enum and the handler's validator both derive from a single ordering var,
// so a published value the handler rejects should be impossible; running the
// whole vocabulary is what keeps that a fact rather than an intention, and it
// catches a tool wired to another entity's vocabulary — the likeliest mistake
// in a change that touches every list tool, since a value valid for the
// intended endpoint is rejected by the other.
func TestOrderingEveryPublishedValueReachesTheWire(t *testing.T) {
	declared := orderingSchemas(t)

	for _, testCase := range orderingToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			properties, ok := declared[testCase.method]
			if !ok {
				t.Fatalf("%s does not declare any ordering parameter", testCase.method)
			}

			call := func(args map[string]any) []url.URL {
				mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK, []byte(`{}`))
				testutil.ExecuteToolRequest(t, mcpServer, testCase.method, args)
				return *urls
			}

			withOrdering := func(extra map[string]any) map[string]any {
				args := maps.Clone(testCase.args)
				if args == nil {
					args = make(map[string]any)
				}
				maps.Copy(args, extra)
				return args
			}

			if testCase.orderModeOnly {
				for _, mode := range []string{"asc", "desc"} {
					urls := call(withOrdering(map[string]any{"order_mode": mode}))
					if got := sentValues(urls, testCase.orderModeParam()); !slices.Contains(got, mode) {
						t.Errorf("expected %s=%q to reach the wire but sent %q (queries: %s)",
							testCase.orderModeParam(), mode, got, rawQueries(urls))
					}
				}
				return
			}

			for _, value := range enumOf(t, properties["order_by"]) {
				urls := call(withOrdering(map[string]any{"order_by": value, "order_mode": "desc"}))

				if got := sentValues(urls, testCase.orderByParam()); !slices.Contains(got, value) {
					t.Errorf("expected %s=%q to reach the wire but sent %q (queries: %s)",
						testCase.orderByParam(), value, got, rawQueries(urls))
				}
				if got := sentValues(urls, testCase.orderModeParam()); !slices.Contains(got, "desc") {
					t.Errorf("expected %s=%q to reach the wire but sent %q (queries: %s)",
						testCase.orderModeParam(), "desc", got, rawQueries(urls))
				}
			}
		})
	}
}

// TestOrderingOmittedKeepsTheExistingOrder pins the decision not to default the
// ordering. Every one of these endpoints documents its own default — tasks sort
// by due date, activities by date — so a tool that filled the parameters in
// would silently reorder the results of every caller that never asked about
// ordering. Defaulting is what twdesk's setPagination does for the Desk list
// endpoints, which is why this is worth pinning on the projects side.
//
// The two sdkDefault fields are the exception, and they are the SDK's doing
// rather than the tool's: NewTasklistBudgetListRequest pre-fills the filters, so
// leaving them untouched sends those values. That is the pre-existing behaviour
// this change preserves — binding the parameters must not clear a filter the
// caller never mentioned.
func TestOrderingOmittedKeepsTheExistingOrder(t *testing.T) {
	for _, testCase := range orderingToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, maps.Clone(testCase.args))

			for param, want := range map[string]string{
				testCase.orderByParam():   testCase.sdkDefaultBy,
				testCase.orderModeParam(): testCase.sdkDefaultMode,
			} {
				got := sentValues(*urls, param)
				switch {
				case want == "" && len(got) > 0:
					t.Errorf("expected no %s in the request query but sent %q (queries: %s)",
						param, got, rawQueries(*urls))
				case want != "" && !slices.Contains(got, want):
					t.Errorf("expected the SDK default %s=%q to survive but sent %q (queries: %s)",
						param, want, got, rawQueries(*urls))
				}
			}
		})
	}
}

// TestOrderingByFieldIDReachesTheWire covers the companion parameter of a
// field-valued order-by. Sorting by a custom field needs two arguments, and the
// API's answer to being given only the first is to ignore the ordering
// altogether, so a companion that never reaches the wire reads as an endpoint
// that does not support the sort at all.
func TestOrderingByFieldIDReachesTheWire(t *testing.T) {
	for _, testCase := range []struct {
		method       string
		args         map[string]any
		orderByValue string
		param        string
		queryParam   string
	}{{
		method:       twprojects.MethodTaskList.String(),
		orderByValue: "customfield",
		param:        "order_by_custom_field_id",
		queryParam:   "orderByCustomFieldId",
	}, {
		method:       twprojects.MethodProjectList.String(),
		orderByValue: "customfield",
		param:        "order_by_custom_field_id",
		queryParam:   "orderByCustomFieldId",
	}, {
		method:       twprojects.MethodCompanyList.String(),
		orderByValue: "customfield",
		param:        "order_by_custom_field_id",
		queryParam:   "orderByCustomFieldId",
	}, {
		method:       twprojects.MethodCustomItemRecordList.String(),
		args:         map[string]any{"custom_item_id": float64(123)},
		orderByValue: "customitemfield",
		param:        "order_by_field_id",
		queryParam:   "orderByFieldId",
	}} {
		t.Run(testCase.method+"/"+testCase.param, func(t *testing.T) {
			args := maps.Clone(testCase.args)
			if args == nil {
				args = make(map[string]any)
			}
			args["order_by"] = testCase.orderByValue
			args[testCase.param] = float64(777)

			mcpServer, urls := testutil.ProjectsMCPServerMockWithRequestURLs(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, testCase.method, args)

			if got := sentValues(*urls, testCase.queryParam); !slices.Contains(got, "777") {
				t.Errorf("expected %s=777 to reach the wire but sent %q (queries: %s)",
					testCase.queryParam, got, rawQueries(*urls))
			}
		})
	}
}

// TestOrderingModeEnumIsAscDesc pins the direction vocabulary every ordering
// tool publishes, so a client can read the two legal values off the schema
// instead of guessing between asc/desc, ASC/DESC and ascending/descending.
func TestOrderingModeEnumIsAscDesc(t *testing.T) {
	declared := orderingSchemas(t)

	for _, testCase := range orderingToolCases {
		t.Run(testCase.method, func(t *testing.T) {
			properties, ok := declared[testCase.method]
			if !ok {
				t.Fatalf("%s does not declare any ordering parameter", testCase.method)
			}
			if want := []string{"asc", "desc"}; !slices.Equal(enumOf(t, properties["order_mode"]), want) {
				t.Errorf("expected the order_mode enum to be %v but got %v",
					want, enumOf(t, properties["order_mode"]))
			}
		})
	}
}
