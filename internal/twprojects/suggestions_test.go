package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/pkg/twctx"
)

// nearMissSearchResponse is what the search endpoint returns for a term that
// names a project while the caller was listing something else.
const nearMissSearchResponse = `{
	"search": [
		{"id": 123, "type": "projects"},
		{"id": 77, "type": "milestones"}
	],
	"included": {
		"projects": {"123": {"id": 123, "name": "Website Redesign"}},
		"milestones": {"77": {"id": 77, "name": "Website Redesign Launch"}}
	}
}`

// nearMissTools is the set of list tools that report near-miss candidates, with
// the attribute each carries its result list under.
var nearMissTools = []struct {
	method  string
	listKey string
}{
	{twprojects.MethodTaskList.String(), "tasks"},
	{twprojects.MethodProjectList.String(), "projects"},
	{twprojects.MethodTasklistList.String(), "tasklists"},
}

// TestListSuggestsNearMissesOnEmptyResult is the main case: a search_term that
// matches nothing in the listed type comes back with the candidates of other
// types, named and typed, and still as a success.
func TestListSuggestsNearMissesOnEmptyResult(t *testing.T) {
	for _, tool := range nearMissTools {
		t.Run(tool.method, func(t *testing.T) {
			mcpServer := nearMissMock(t, tool.listKey, "[]", nearMissSearchResponse)

			testutil.ExecuteToolRequest(t, mcpServer, tool.method, map[string]any{
				"search_term": "Website Redesign",
			}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
				t.Helper()
				// An empty result with suggestions is not an error.
				testutil.CheckMessage(t, result)

				want := []map[string]any{
					{"type": "project", "id": float64(123), "name": "Website Redesign"},
					{"type": "milestone", "id": float64(77), "name": "Website Redesign Launch"},
				}
				if got := suggestionsFromToolResult(t, result); !reflect.DeepEqual(got, want) {
					t.Errorf("expected suggestions %v but got %v", want, got)
				}
			}))
		})
	}
}

// TestListSuggestionsRequestUnrestrictedSearch guards the request the lookup
// sends: the caller's term, no type filter — the listed type is the one that
// just came back empty — and the sideloads needed to name a hit. The mock
// answers both calls with the same body either way, so only the query string
// can show it.
func TestListSuggestionsRequestUnrestrictedSearch(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "search.json", Status: http.StatusOK, Body: []byte(nearMissSearchResponse)},
	}, http.StatusOK, []byte(`{"tasks": []}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term": "Website Redesign",
	})

	search := searchRequestFromRecording(t, *recorded)
	query := search.URL.Query()
	if got := query.Get("searchTerm"); got != "Website Redesign" {
		t.Errorf("expected searchTerm=%q but got %q", "Website Redesign", got)
	}
	if query.Has("type") {
		t.Errorf("expected no type filter on the suggestion search but got %q", query.Get("type"))
	}
	for _, want := range []string{"projects", "tasks", "tasklists", "milestones", "users"} {
		if !strings.Contains(query.Get("include"), want) {
			t.Errorf("expected the %q sideload in include=%q", want, query.Get("include"))
		}
	}
	if got := query.Get("fields[projects]"); got != "id,name" {
		t.Errorf("expected fields[projects]=%q but got %q", "id,name", got)
	}
	// Completed items are asked for unconditionally, not inherited: the list that
	// hid them is the one that came back empty.
	if got := query.Get("includeCompletedItems"); got != "true" {
		t.Errorf("expected includeCompletedItems=%q but got %q", "true", got)
	}
}

// TestListSuggestionsIgnoreShowCompleted pins that the lookup does not inherit
// the caller's completion filter. A caller listing only open work is the one
// most likely to get an empty array for a name that does exist.
func TestListSuggestionsIgnoreShowCompleted(t *testing.T) {
	for _, showCompleted := range []bool{false, true} {
		t.Run(strconv.FormatBool(showCompleted), func(t *testing.T) {
			mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
				{Match: "search.json", Status: http.StatusOK, Body: []byte(nearMissSearchResponse)},
			}, http.StatusOK, []byte(`{"tasks": []}`))

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
				"search_term":    "Website Redesign",
				"show_completed": showCompleted,
			})

			search := searchRequestFromRecording(t, *recorded)
			if got := search.URL.Query().Get("includeCompletedItems"); got != "true" {
				t.Errorf("expected includeCompletedItems=%q with show_completed=%v but got %q",
					"true", showCompleted, got)
			}
		})
	}
}

// TestListSuggestionsCanNameTheListedType covers the consequence of searching
// completed items: the candidate that explains an empty result is often the same
// type the caller listed, hidden by its own filters rather than living under
// another type.
func TestListSuggestionsCanNameTheListedType(t *testing.T) {
	search := `{
		"search": [{"id": 42, "type": "tasks"}],
		"included": {"tasks": {"42": {"id": 42, "name": "Ship the redesign"}}}
	}`

	mcpServer := nearMissMock(t, "tasks", "[]", search)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term": "Ship the redesign",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		want := []map[string]any{{"type": "task", "id": float64(42), "name": "Ship the redesign"}}
		if got := suggestionsFromToolResult(t, result); !reflect.DeepEqual(got, want) {
			t.Errorf("expected suggestions %v but got %v", want, got)
		}
	}))
}

// TestListSkipsSuggestionsWhenResultsFound pins that a non-empty list never
// pays for the extra request.
func TestListSkipsSuggestionsWhenResultsFound(t *testing.T) {
	for _, tool := range nearMissTools {
		t.Run(tool.method, func(t *testing.T) {
			list := `{"` + tool.listKey + `": [{"id": 1, "name": "Website Redesign"}]}`
			mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
				{Match: "search.json", Status: http.StatusOK, Body: []byte(nearMissSearchResponse)},
			}, http.StatusOK, []byte(list))

			testutil.ExecuteToolRequest(t, mcpServer, tool.method, map[string]any{
				"search_term": "Website Redesign",
			}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
				t.Helper()
				testutil.CheckMessage(t, result)

				if got := suggestionsFromToolResult(t, result); got != nil {
					t.Errorf("expected no suggestions alongside results but got %v", got)
				}
			}))

			assertNoSuggestionSearch(t, *recorded)
		})
	}
}

// TestListSkipsSuggestionsWithoutSearchTerm pins that the lookup is tied to the
// term. An empty list under structured filters alone says the filters matched
// nothing, which no name can explain.
func TestListSkipsSuggestionsWithoutSearchTerm(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "search.json", Status: http.StatusOK, Body: []byte(nearMissSearchResponse)},
	}, http.StatusOK, []byte(`{"tasks": []}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"project_id": float64(777),
	})

	assertNoSuggestionSearch(t, *recorded)
}

// TestListSkipsSuggestionsForShortSearchTerm guards the length floor: the
// search endpoint rejects a term under three characters with a 400, so sending
// one would spend a request to learn nothing.
func TestListSkipsSuggestionsForShortSearchTerm(t *testing.T) {
	mcpServer, recorded := testutil.ProjectsMCPServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Match: "search.json", Status: http.StatusOK, Body: []byte(nearMissSearchResponse)},
	}, http.StatusOK, []byte(`{"tasks": []}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term": "ab",
	})

	assertNoSuggestionSearch(t, *recorded)
}

// TestListSuggestionsAreBestEffort covers a failing lookup: the caller keeps
// the empty result it asked for, as a success, with no suggestions.
func TestListSuggestionsAreBestEffort(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			// The override route takes precedence over the search route
			// nearMissMock appends, so the search fails while the list succeeds.
			mcpServer := nearMissMock(t, "tasks", "[]", nearMissSearchResponse,
				testutil.ProjectsMockRoute{Match: "search.json", Status: status, Body: []byte(`{"error": "nope"}`)})

			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
				"search_term": "Website Redesign",
			}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
				t.Helper()
				// The list request succeeded; a failed suggestion lookup must not
				// turn that into an error.
				testutil.CheckMessage(t, result)

				if got := suggestionsFromToolResult(t, result); got != nil {
					t.Errorf("expected no suggestions after a %d from the search but got %v", status, got)
				}
			}))
		})
	}
}

// TestListSuggestionsCapAtFive pins the cap, and that it counts named
// candidates rather than hits: the lookup keeps scanning past a hit it cannot
// name instead of spending a slot on it.
func TestListSuggestionsCapAtFive(t *testing.T) {
	var hits []string
	tasks := map[string]any{}
	for id := 1; id <= 8; id++ {
		key := strconv.Itoa(id)
		hits = append(hits, `{"id": `+key+`, "type": "tasks"}`)
		tasks[key] = map[string]any{"id": id, "name": "Task " + key}
	}
	// A comment hit has no name to report, so it is skipped rather than counted.
	hits = append([]string{`{"id": 99, "type": "comments"}`}, hits...)
	encodedTasks, err := json.Marshal(tasks)
	if err != nil {
		t.Fatalf("failed to encode sideload: %v", err)
	}
	search := `{"search": [` + strings.Join(hits, ",") + `],
		"included": {"tasks": ` + string(encodedTasks) + `,
		"comments": {"99": {"id": 99, "title": "a comment mentioning it"}}}}`

	mcpServer := nearMissMock(t, "tasklists", "[]", search)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistList.String(), map[string]any{
		"search_term": "Task",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		got := suggestionsFromToolResult(t, result)
		if len(got) != 5 {
			t.Fatalf("expected 5 suggestions but got %d: %v", len(got), got)
		}
		for i, suggestion := range got {
			if suggestion["type"] != "task" {
				t.Errorf("expected suggestion %d to be a task but got %v", i, suggestion["type"])
			}
			if suggestion["name"] != "Task "+strconv.Itoa(i+1) {
				t.Errorf("expected suggestion %d to keep relevance order, got %v", i, suggestion["name"])
			}
		}
	}))
}

// TestListSuggestionsSkipUnnamedHits covers the case where nothing in the
// search can be named: no suggestions key at all, rather than entries the model
// cannot act on.
func TestListSuggestionsSkipUnnamedHits(t *testing.T) {
	search := `{
		"search": [{"id": 11, "type": "comments"}, {"id": 22, "type": "timelogs"}, {"id": 33, "type": "tasks"}],
		"included": {
			"comments": {"11": {"id": 11, "title": "mentions the term"}},
			"timelogs": {"22": {"id": 22, "description": "mentions the term"}},
			"tasks": {}
		}
	}`

	mcpServer := nearMissMock(t, "projects", "[]", search)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), map[string]any{
		"search_term": "the term",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		if got := suggestionsFromToolResult(t, result); got != nil {
			t.Errorf("expected no suggestions for hits that carry no name but got %v", got)
		}
	}))
}

// TestListSuggestionsCarryNoWebLink guards the ordering against WebLinker: a
// suggestion added before the linker ran would be stamped with a link built
// from the calling tool's own path prefix, pointing a project candidate at
// /app/tasks/123.
//
// Both halves need the customer URL in context, which the projects mocks do not
// set: without it WebLinker returns the body untouched and the assertion passes
// whatever the order. The first half is the control that proves the linker is
// live here.
func TestListSuggestionsCarryNoWebLink(t *testing.T) {
	t.Run("linker is active", func(t *testing.T) {
		mcpServer := withCustomerURL(nearMissMock(t, "tasks", `[{"id": 5, "name": "a task"}]`, nearMissSearchResponse))

		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
			"search_term": "Website Redesign",
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			var payload struct {
				Tasks []struct {
					Meta struct {
						WebLink string `json:"webLink"`
					} `json:"meta"`
				} `json:"tasks"`
			}
			if err := json.Unmarshal([]byte(searchTextFromToolResult(t, result)), &payload); err != nil {
				t.Fatalf("failed to decode tool output: %v", err)
			}
			if len(payload.Tasks) != 1 {
				t.Fatalf("expected a single task but got %d", len(payload.Tasks))
			}
			if want := customerURL + "/app/tasks/5"; payload.Tasks[0].Meta.WebLink != want {
				t.Fatalf("expected the linker to stamp %q but got %q", want, payload.Tasks[0].Meta.WebLink)
			}
		}))
	})

	t.Run("suggestions are not linked", func(t *testing.T) {
		mcpServer := withCustomerURL(nearMissMock(t, "tasks", "[]", nearMissSearchResponse))

		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
			"search_term": "Website Redesign",
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			suggestions := suggestionsFromToolResult(t, result)
			if len(suggestions) == 0 {
				t.Fatal("expected suggestions to assert on")
			}
			for _, suggestion := range suggestions {
				if _, ok := suggestion["meta"]; ok {
					t.Errorf("expected no injected meta on a suggestion but got %v", suggestion["meta"])
				}
			}
		}))
	})
}

// TestListSuggestionsInStructuredContent pins that the suggestions reach the
// structured content too, not only the text body.
func TestListSuggestionsInStructuredContent(t *testing.T) {
	mcpServer := nearMissMock(t, "tasks", "[]", nearMissSearchResponse)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskList.String(), map[string]any{
		"search_term": "Website Redesign",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		structured, ok := toolResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("unexpected structured content type: %T", toolResult.StructuredContent)
		}
		suggestions, ok := structured["suggestions"].([]any)
		if !ok || len(suggestions) != 2 {
			t.Fatalf("expected 2 suggestions in the structured content but got %v", structured["suggestions"])
		}
	}))
}

// TestListSuggestionsValidateAgainstOutputSchema checks a suggestion-bearing
// response against the tool's published output schema by hand: neither the
// server nor ExecuteToolRequest validates structured content, so a schema that
// does not describe the suggestions would ship silently and fail at every
// validating client.
func TestListSuggestionsValidateAgainstOutputSchema(t *testing.T) {
	for _, tool := range nearMissTools {
		t.Run(tool.method, func(t *testing.T) {
			mcpServer := nearMissMock(t, tool.listKey, "[]", nearMissSearchResponse)

			testutil.ExecuteToolRequest(t, mcpServer, tool.method, map[string]any{
				"search_term": "Website Redesign",
			}, testutil.ExecuteToolRequestWithCheckMessage(
				checkStructuredContentMatchesOutputSchema(tool.method),
			))
		})
	}
}

// customerURL is the site the web-link tests pretend to run against.
const customerURL = "https://example.teamwork.com"

// withCustomerURL puts the customer URL in the request context, which is what
// WebLinker needs to inject a link at all. The projects mocks leave it unset, so
// any test that asserts on linking has to add it.
func withCustomerURL(mcpServer *mcp.Server) *mcp.Server {
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			return next(twctx.WithCustomerURL(ctx, customerURL), method, req)
		}
	})
	return mcpServer
}

// nearMissMock builds a server whose list endpoint answers with the given
// result list and whose search endpoint answers with the given body. Extra
// routes take precedence, so a test can override the search response.
func nearMissMock(t *testing.T, listKey, list, search string, routes ...testutil.ProjectsMockRoute) *mcp.Server {
	t.Helper()

	routes = append(routes, testutil.ProjectsMockRoute{
		Match:  "search.json",
		Status: http.StatusOK,
		Body:   []byte(search),
	})
	return testutil.ProjectsMCPServerRoutedMock(t, routes, http.StatusOK,
		[]byte(`{"`+listKey+`": `+list+`}`))
}

// searchRequestFromRecording returns the single request the tool sent to the
// search endpoint.
func searchRequestFromRecording(t *testing.T, recorded []testutil.ProjectsRecordedRequest) testutil.ProjectsRecordedRequest {
	t.Helper()

	var searches []testutil.ProjectsRecordedRequest
	for _, request := range recorded {
		if strings.Contains(request.URL.Path, "search.json") {
			searches = append(searches, request)
		}
	}
	if len(searches) != 1 {
		t.Fatalf("expected exactly one search request but got %d out of %d requests", len(searches), len(recorded))
	}
	return searches[0]
}

// assertNoSuggestionSearch fails when the tool reached the search endpoint.
func assertNoSuggestionSearch(t *testing.T, recorded []testutil.ProjectsRecordedRequest) {
	t.Helper()

	for _, request := range recorded {
		if strings.Contains(request.URL.Path, "search.json") {
			t.Errorf("unexpected suggestion search request to %s", request.URL.String())
		}
	}
}

// suggestionsFromToolResult decodes the suggestions out of a list tool result,
// returning nil when the attribute is absent.
func suggestionsFromToolResult(t *testing.T, result mcp.Result) []map[string]any {
	t.Helper()

	var payload struct {
		Suggestions []map[string]any `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(searchTextFromToolResult(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	return payload.Suggestions
}
