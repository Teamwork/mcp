package twprojects_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	pkgtestutil "github.com/teamwork/mcp/pkg/testutil"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// bulkTimelogRequest is one request a bulk timelog tool sent.
type bulkTimelogRequest struct {
	method string
	path   string
	body   map[string]any
}

// bulkTimelogMock answers every request with the given status, headers and
// body, and records each request, so a test can assert both the one request a
// bulk tool sends and that it sent no more than one. The shared mocks answer
// without headers, and set_timelogs_billable reads its count from one.
func bulkTimelogMock(
	t *testing.T,
	status int,
	headers map[string]string,
	response string,
) (*mcp.Server, *[]bulkTimelogRequest) {
	t.Helper()

	var requests []bulkTimelogRequest
	engine := twapi.NewEngine(testutil.ProjectsSessionMock{},
		twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				recorded := bulkTimelogRequest{method: req.Method, path: req.URL.Path}
				if req.Body != nil {
					raw, err := io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					if err := json.Unmarshal(raw, &recorded.body); err != nil {
						t.Errorf("failed to decode request body %q: %s", raw, err)
					}
				}
				requests = append(requests, recorded)
				resp := pkgtestutil.NewMockHTTPResponse(status, []byte(response))
				for key, value := range headers {
					resp.Header.Set(key, value)
				}
				return resp, nil
			})
		}),
	)
	return pkgtestutil.MCPServer(t, twprojects.DefaultToolsetGroup(false, true, engine)), &requests
}

// bulkResult returns the text and error flag of a tool result.
func bulkResult(t *testing.T, result mcp.Result) (string, bool) {
	t.Helper()
	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	var texts []string
	for _, content := range toolResult.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			texts = append(texts, text.Text)
		}
	}
	return strings.Join(texts, "\n"), toolResult.IsError
}

// assertOneBulkRequest checks that exactly one request was sent, with the given
// method, path and body.
func assertOneBulkRequest(
	t *testing.T,
	requests []bulkTimelogRequest,
	method, path string,
	body map[string]any,
) {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("expected exactly one request, got %d: %v", len(requests), requests)
	}
	request := requests[0]
	if request.method != method {
		t.Errorf("expected method %s, got %s", method, request.method)
	}
	if request.path != path {
		t.Errorf("expected path %s, got %s", path, request.path)
	}
	if len(request.body) != len(body) {
		t.Errorf("expected body %v, got %v", body, request.body)
	}
	for key, want := range body {
		if got := request.body[key]; got != want {
			t.Errorf("expected %s to be %v, got %v", key, want, got)
		}
	}
}

func TestTimelogBulkMoveSendsOneRequest(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusOK, nil, `{"STATUS":"OK","numberOfMovedTimelogs":"3"}`)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBulkMove.String(), map[string]any{
		"source_project_id": float64(777),
		"target_project_id": float64(778),
		"timelog_ids":       []any{float64(1), float64(2), float64(3), float64(2)},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		text, isError := bulkResult(t, result)
		if isError {
			t.Fatalf("expected success, got %s", text)
		}
		if !strings.Contains(text, "Moved 3 timelogs") {
			t.Errorf("expected the moved count in the result, got %q", text)
		}
	}))

	// a duplicate ID is sent once
	assertOneBulkRequest(t, *requests, http.MethodPut, "/projects/777/time_entries/move.json", map[string]any{
		"projectId":     float64(778),
		"timelogIdList": "1,2,3",
	})
}

// TestTimelogBulkMovePartialSuccess pins the result when the endpoint skipped
// some of the listed timelogs: it reports a count only, so the result says how
// many were skipped and that it cannot say which.
func TestTimelogBulkMovePartialSuccess(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusOK, nil, `{"STATUS":"OK","numberOfMovedTimelogs":"2"}`)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBulkMove.String(), map[string]any{
		"source_project_id": float64(777),
		"target_project_id": float64(778),
		"timelog_ids":       []any{float64(1), float64(2), float64(3)},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		text, isError := bulkResult(t, result)
		if isError {
			t.Fatalf("expected success, got %s", text)
		}
		for _, want := range []string{"Moved 2 of 3", "other 1 were skipped", "does not say which"} {
			if !strings.Contains(text, want) {
				t.Errorf("expected %q in the result, got %q", want, text)
			}
		}
	}))
	if len(*requests) != 1 {
		t.Errorf("expected exactly one request, got %d", len(*requests))
	}
}

// TestTimelogBulkMoveNothingMoved pins that the endpoint's 200 errMsg answer,
// which it sends when none of the listed timelogs could be moved, is an error
// result rather than a move.
func TestTimelogBulkMoveNothingMoved(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusOK, nil, `{"errMsg":"No valid time logs to move"}`)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBulkMove.String(), map[string]any{
		"source_project_id": float64(777),
		"target_project_id": float64(778),
		"timelog_ids":       []any{float64(1)},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		text, isError := bulkResult(t, result)
		if !isError {
			t.Fatalf("expected an error result, got %q", text)
		}
		for _, want := range []string{"No timelogs were moved", "No valid time logs to move"} {
			if !strings.Contains(text, want) {
				t.Errorf("expected %q in the result, got %q", want, text)
			}
		}
	}))
	if len(*requests) != 1 {
		t.Errorf("expected exactly one request, got %d", len(*requests))
	}
}

// TestTimelogBulkMoveRejectsBeforeSending pins the checks that refuse a call
// without sending anything: a list over the limit, an empty list, and a move
// to the same project.
func TestTimelogBulkMoveRejectsBeforeSending(t *testing.T) {
	tooMany := make([]any, 101)
	for i := range tooMany {
		tooMany[i] = float64(i + 1)
	}

	tests := []struct {
		name      string
		arguments map[string]any
		want      string
	}{{
		name: "too many timelogs",
		arguments: map[string]any{
			"source_project_id": float64(777),
			"target_project_id": float64(778),
			"timelog_ids":       tooMany,
		},
		want: "100",
	}, {
		name: "same project",
		arguments: map[string]any{
			"source_project_id": float64(777),
			"target_project_id": float64(777),
			"timelog_ids":       []any{float64(1)},
		},
		want: "must differ",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requests := bulkTimelogMock(t, http.StatusOK, nil, `{"STATUS":"OK","numberOfMovedTimelogs":"1"}`)
			// The published schema caps the list and the handler checks it again;
			// either way the call is refused before anything is sent.
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBulkMove.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					text, isError := bulkResult(t, result)
					if !isError {
						t.Fatalf("expected an error result, got %q", text)
					}
					if !strings.Contains(text, tt.want) {
						t.Errorf("expected %q in the result, got %q", tt.want, text)
					}
				}))
			if len(*requests) != 0 {
				t.Errorf("expected no request to be sent, got %d", len(*requests))
			}
		})
	}
}

func TestTimelogBillableBulkUpdateSendsOneRequest(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		body      map[string]any
		want      string
	}{{
		name: "make billable, every filter",
		arguments: map[string]any{
			"project_id":     float64(777),
			"billable":       true,
			"start_date":     "2026-08-01",
			"end_date":       "2026-08-31",
			"user_ids":       []any{float64(12345)},
			"tag_ids":        []any{float64(10), float64(11)},
			"match_all_tags": true,
		},
		body: map[string]any{
			"isBillable":   true,
			"projectId":    "777",
			"fromDate":     "20260801",
			"toDate":       "20260831",
			"userId":       "12345",
			"tagIds":       "10,11",
			"matchAllTags": true,
			"billableType": "nonbillable",
			"invoicedType": "noninvoiced",
		},
		want: "Changed 4 timelogs in project 777 to billable",
	}, {
		name: "make non-billable, invoiced included",
		arguments: map[string]any{
			"project_id":       float64(777),
			"billable":         false,
			"include_invoiced": true,
		},
		body: map[string]any{
			"isBillable":   false,
			"projectId":    "777",
			"billableType": "billable",
			"invoicedType": "all",
		},
		want: "Changed 4 timelogs in project 777 to non-billable",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requests := bulkTimelogMock(t, http.StatusOK, map[string]string{"count": "4"}, `{"STATUS":"OK"}`)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBillableBulk.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					text, isError := bulkResult(t, result)
					if isError {
						t.Fatalf("expected success, got %s", text)
					}
					if !strings.Contains(text, tt.want) {
						t.Errorf("expected %q in the result, got %q", tt.want, text)
					}
				}))
			assertOneBulkRequest(t, *requests, http.MethodPut, "/time_entries.json", tt.body)
		})
	}
}

// TestTimelogBillableBulkUpdateMatchesNothing pins the endpoint's 422 for a
// filter matching no timelog: nothing changed, which is a count of zero rather
// than a failure.
func TestTimelogBillableBulkUpdateMatchesNothing(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusUnprocessableEntity, nil,
		`{"STATUS":"Error","MESSAGE":"Nothing to modify based on selected options"}`)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBillableBulk.String(), map[string]any{
		"project_id": float64(777),
		"billable":   true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		text, isError := bulkResult(t, result)
		if isError {
			t.Fatalf("expected a non-error result, got %s", text)
		}
		for _, want := range []string{"Changed 0 timelogs", "Nothing was changed"} {
			if !strings.Contains(text, want) {
				t.Errorf("expected %q in the result, got %q", want, text)
			}
		}
	}))
	if len(*requests) != 1 {
		t.Errorf("expected exactly one request, got %d", len(*requests))
	}
}

// TestTimelogBillableBulkUpdateForbidden pins that a refusal is reported as an
// error result, not as zero changes.
func TestTimelogBillableBulkUpdateForbidden(t *testing.T) {
	mcpServer, _ := bulkTimelogMock(t, http.StatusForbidden, nil, `{"STATUS":"Error","MESSAGE":"Forbidden"}`)

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBillableBulk.String(), map[string]any{
		"project_id": float64(777),
		"billable":   true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		if text, isError := bulkResult(t, result); !isError {
			t.Fatalf("expected an error result, got %q", text)
		}
	}))
}

// TestTimelogBillableBulkUpdateRefusesAWideFilter pins that the tool never
// sends a filter without a project: the endpoint reads a missing or zero
// project as every project in the installation.
func TestTimelogBillableBulkUpdateRefusesAWideFilter(t *testing.T) {
	tooMany := make([]any, 51)
	for i := range tooMany {
		tooMany[i] = float64(i + 1)
	}

	tests := []struct {
		name      string
		arguments map[string]any
		want      string
	}{{
		name:      "zero project",
		arguments: map[string]any{"project_id": float64(0), "billable": true},
		want:      "project_id must be a positive",
	}, {
		name:      "too many users",
		arguments: map[string]any{"project_id": float64(777), "billable": true, "user_ids": tooMany},
		want:      "50",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requests := bulkTimelogMock(t, http.StatusOK, map[string]string{"count": "1"}, `{"STATUS":"OK"}`)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBillableBulk.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					text, isError := bulkResult(t, result)
					if !isError {
						t.Fatalf("expected an error result, got %q", text)
					}
					if !strings.Contains(text, tt.want) {
						t.Errorf("expected %q in the result, got %q", tt.want, text)
					}
				}))
			if len(*requests) != 0 {
				t.Errorf("expected no request to be sent, got %d", len(*requests))
			}
		})
	}

	// project_id is required by the published schema, so a call without one is
	// rejected before the handler runs.
	mcpServer, requests := bulkTimelogMock(t, http.StatusOK, nil, `{"STATUS":"OK"}`)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogBillableBulk.String(),
		map[string]any{"billable": true},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			if text, isError := bulkResult(t, result); !isError {
				t.Fatalf("expected an error result, got %q", text)
			}
		}))
	if len(*requests) != 0 {
		t.Errorf("expected no request to be sent, got %d", len(*requests))
	}
}

// timelogBatchItem is a valid create_timelogs item against the given target.
func timelogBatchItem(target string, id float64) map[string]any {
	return map[string]any{target: id, "date": "2026-10-20", "time": "09:30:00", "hours": float64(1), "minutes": float64(0)}
}

// singleTimelogBulkCreateEntries asserts that exactly one request reached the
// bulk route with partial success on, and returns its entries' timelogs.
func singleTimelogBulkCreateEntries(t *testing.T, requests []bulkTimelogRequest) []map[string]any {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("expected exactly 1 request, got %d", len(requests))
	}
	request := requests[0]
	if request.method != http.MethodPost || request.path != "/projects/api/v3/time/bulk.json" {
		t.Errorf("expected POST /projects/api/v3/time/bulk.json, got %s %s", request.method, request.path)
	}
	if options, _ := request.body["timelogOptions"].(map[string]any); options["partialSuccess"] != true {
		t.Errorf("expected timelogOptions.partialSuccess true, got %v", request.body["timelogOptions"])
	}
	raw, _ := request.body["timelogs"].([]any)
	entries := make([]map[string]any, len(raw))
	for i, entry := range raw {
		entries[i], _ = entry.(map[string]any)["timelog"].(map[string]any)
	}
	return entries
}

// TestTimelogCreateBatchSendsOneRequest pins a full success: every item goes in
// one request with its own fields, and each is reported with its new ID.
func TestTimelogCreateBatchSendsOneRequest(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusCreated, nil, `{"timelogs":[{"id":101},{"id":102}]}`)

	second := timelogBatchItem("task_id", 12345)
	second["description"] = "Review"
	second["is_utc"] = true
	second["billable"] = false
	second["user_id"] = float64(777)
	second["tag_ids"] = []any{float64(1), float64(2)}
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreateBatch.String(), map[string]any{
		"timelogs": []any{timelogBatchItem("project_id", 777), second},
	}, expectBatchResult(false, "Created 2 of 2 timelogs",
		"item 1 (project 777, 2026-10-20) → timelog 101", "item 2 (task 12345, 2026-10-20) → timelog 102"))

	entries := singleTimelogBulkCreateEntries(t, *requests)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	first := entries[0]
	if first["projectId"] != float64(777) || first["date"] != "2026-10-20" || first["time"] != "09:30:00" ||
		first["hours"] != float64(1) {
		t.Errorf("unexpected first entry: %v", first)
	}
	if _, ok := first["taskId"]; ok {
		t.Errorf("expected no taskId on a project entry: %v", first)
	}
	for key, want := range map[string]any{
		"taskId": float64(12345), "description": "Review", "isUTC": true, "isBillable": false, "userId": float64(777),
	} {
		if entries[1][key] != want {
			t.Errorf("second entry: expected %s=%v, got %v", key, want, entries[1][key])
		}
	}
	if tags, _ := entries[1]["tagIds"].([]any); len(tags) != 2 {
		t.Errorf("second entry: expected 2 tagIds, got %v", entries[1]["tagIds"])
	}
}

// TestTimelogCreateBatchReportsEachFailure maps a partial success back to the
// items: the created timelogs take the indexes not reported as failed, in order.
func TestTimelogCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, requests := bulkTimelogMock(t, http.StatusCreated, nil, `{
		"timelogs":[{"id":101},{"id":103}],
		"failed":[{"index":1,"errors":[{"title":"invalid payload","detail":"a task is required"}]}]
	}`)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreateBatch.String(), map[string]any{
		"timelogs": []any{
			timelogBatchItem("task_id", 1), timelogBatchItem("project_id", 777), timelogBatchItem("task_id", 3),
		},
	}, expectBatchResult(true, "Created 2 of 3 timelogs",
		"item 1 (task 1, 2026-10-20) → timelog 101",
		"item 3 (task 3, 2026-10-20) → timelog 103",
		"Failed: item 2 (project 777, 2026-10-20): a task is required"))

	singleTimelogBulkCreateEntries(t, *requests)
}

// TestTimelogCreateBatchNothingCreated reports a request rejected as a whole,
// naming each item it blamed, for both statuses the endpoint uses.
func TestTimelogCreateBatchNothingCreated(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			mcpServer, requests := bulkTimelogMock(t, status, nil, `{"errors":[
				{"title":"invalid payload","detail":"project not found","meta":{"index":0}},
				{"title":"forbidden action","detail":"cannot log time","meta":{"index":1}}
			]}`)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreateBatch.String(), map[string]any{
				"timelogs": []any{timelogBatchItem("project_id", 1), timelogBatchItem("task_id", 2)},
			}, expectBatchResult(true, "Created 0 of 2 timelogs. Nothing was created.",
				"Failed: item 1 (project 1, 2026-10-20): project not found",
				"Failed: item 2 (task 2, 2026-10-20): cannot log time"))

			singleTimelogBulkCreateEntries(t, *requests)
		})
	}
}

// TestTimelogCreateBatchServerErrorSaysToCheck keeps a 5xx from reading as
// "nothing was created": entries are committed one by one.
func TestTimelogCreateBatchServerErrorSaysToCheck(t *testing.T) {
	mcpServer, _ := bulkTimelogMock(t, http.StatusInternalServerError, nil, `{"errors":[{"title":"internal"}]}`)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreateBatch.String(), map[string]any{
		"timelogs": []any{timelogBatchItem("task_id", 1)},
	}, expectBatchResult(true, "may have been created", twprojects.MethodTimelogList.String()))
}

// TestTimelogCreateBatchLostAnswerSaysToCheck covers a request whose answer
// never arrived, such as a timeout: the outcome is unknown, so the result says
// to check before resending, and it is a tool result rather than a Go error.
func TestTimelogCreateBatchLostAnswerSaysToCheck(t *testing.T) {
	engine := twapi.NewEngine(testutil.ProjectsSessionMock{},
		twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			})
		}),
	)
	arguments, err := json.Marshal(map[string]any{"timelogs": []any{timelogBatchItem("task_id", 1)}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := twprojects.TimelogCreateBatch(engine).Handler(t.Context(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: twprojects.MethodTimelogCreateBatch.String(), Arguments: arguments},
	})
	if err != nil {
		t.Fatalf("expected a tool result rather than a Go error: %v", err)
	}
	text, isError := bulkResult(t, result)
	if !isError || !strings.Contains(text, "may have been created") ||
		!strings.Contains(text, twprojects.MethodTimelogList.String()) {
		t.Errorf("expected an error result saying to check, got: %s", text)
	}
}

// TestTimelogCreateBatchRejectsAnInvalidItemBeforeSending keeps one bad item
// from sending the rest, including an item naming both or neither target.
func TestTimelogCreateBatchRejectsAnInvalidItemBeforeSending(t *testing.T) {
	both := timelogBatchItem("task_id", 1)
	both["project_id"] = float64(777)
	neither := timelogBatchItem("task_id", 1)
	delete(neither, "task_id")

	mcpServer, requests := bulkTimelogMock(t, http.StatusCreated, nil, `{"timelogs":[{"id":1}]}`)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimelogCreateBatch.String(), map[string]any{
		"timelogs": []any{timelogBatchItem("task_id", 1), both, neither},
	}, expectBatchResult(true, "Nothing was written",
		"item 2: provide exactly one of project_id or task_id",
		"item 3: provide exactly one of project_id or task_id"))

	if len(*requests) != 0 {
		t.Errorf("expected no requests, got %d", len(*requests))
	}
}

// TestTimelogCreateBatchRejectsATooLongList refuses a list over the maximum in
// the handler too, for clients that skip schema validation.
func TestTimelogCreateBatchRejectsATooLongList(t *testing.T) {
	timelogs := make([]any, 101)
	for i := range timelogs {
		timelogs[i] = timelogBatchItem("task_id", 1)
	}
	arguments, err := json.Marshal(map[string]any{"timelogs": timelogs})
	if err != nil {
		t.Fatal(err)
	}
	result, err := twprojects.TimelogCreateBatch(nil).Handler(t.Context(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: twprojects.MethodTimelogCreateBatch.String(), Arguments: arguments},
	})
	if err != nil {
		t.Fatalf("expected a tool result rather than a Go error: %v", err)
	}
	if text, isError := bulkResult(t, result); !isError || !strings.Contains(text, "at most 100 items, got 101") {
		t.Errorf("expected the list to be refused, got: %s", text)
	}
}
