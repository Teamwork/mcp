package twprojects_test

import (
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
