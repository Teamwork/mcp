package twprojects_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// allocationBody is a response row carrying the date and date-time shapes the
// mocks otherwise leave nil: startedAt/endedAt are twapi.Date, deletedAt is a
// set *time.Time. A mock replying `{}` exercises neither, so a schema describing
// twapi.Date by time.Time's unexported fields would pass every other test here.
const allocationBody = `{
	"id": 12345,
	"title": "Design phase",
	"startedAt": "2026-09-01",
	"endedAt": "2026-09-30",
	"createdAt": "2026-08-01T09:00:00Z",
	"deletedAt": "2026-08-19T17:30:00Z",
	"project": {"id": 777, "type": "projects"},
	"assignedUser": {"id": 456, "type": "users"},
	"linkedTaskIDs": [{"id": 99, "type": "tasks"}],
	"duration": 4800,
	"color": "3c8f7c",
	"secondsPerDay": 14400,
	"status": "active"
}`

func TestAllocationCreate(t *testing.T) {
	mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"allocation":{"id":12345}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreate.String(), map[string]any{
		"project_id":        float64(777),
		"assigned_user_id":  float64(456),
		"title":             "Design phase",
		"start_date":        "2026-09-01",
		"end_date":          "2026-09-30",
		"seconds_per_day":   float64(14400),
		"color":             "#3c8f7c",
		"description":       "Four hours a day",
		"is_billable":       true,
		"ignore_collisions": true,
		"linked_task_ids":   []float64{1, 2},
	})

	var payload struct {
		Allocation map[string]any `json:"allocation"`
	}
	if err := json.Unmarshal(*body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %s", err)
	}
	want := map[string]any{
		"projectId":              float64(777),
		"assignedUserID":         float64(456),
		"title":                  "Design phase",
		"startedAt":              "2026-09-01",
		"endedAt":                "2026-09-30",
		"secondsPerDay":          float64(14400),
		"color":                  "#3c8f7c",
		"description":            "Four hours a day",
		"isBillable":             true,
		"ignoreCollisions":       true,
		"informOfOverAllocation": true,
	}
	for key, expected := range want {
		if got := payload.Allocation[key]; got != expected {
			t.Errorf("expected %s to be %v, got %v", key, expected, got)
		}
	}
	if got := payload.Allocation["linkedTaskIDs"]; len(got.([]any)) != 2 {
		t.Errorf("expected two linked task ids, got %v", got)
	}
}

// TestAllocationCreateOmitsUnsetOptionals pins that an optional the caller left
// out is absent from the payload rather than sent as a zero value. Every
// writable field is a pointer for exactly this reason, and a plain value would
// look identical in a test that only asserts the fields it did send.
func TestAllocationCreateOmitsUnsetOptionals(t *testing.T) {
	mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"allocation":{"id":12345}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreate.String(), map[string]any{
		"project_id":       float64(777),
		"assigned_user_id": float64(456),
		"title":            "Design phase",
		"start_date":       "2026-09-01",
		"end_date":         "2026-09-30",
		"seconds_per_day":  float64(14400),
		"color":            "#3c8f7c",
	})

	var payload struct {
		Allocation map[string]any `json:"allocation"`
	}
	if err := json.Unmarshal(*body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %s", err)
	}
	for _, key := range []string{
		"description", "isBillable", "ignoreCollisions", "linkedTaskIDs", "duration", "hoursPerDay",
		"distributeType", "recurringRule",
	} {
		if _, ok := payload.Allocation[key]; ok {
			t.Errorf("expected %s to be absent from the payload, got %v", key, payload.Allocation[key])
		}
	}
}

func TestAllocationUpdate(t *testing.T) {
	mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusOK,
		[]byte(`{"allocation":{"id":12345}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationUpdate.String(), map[string]any{
		"id":              float64(12345),
		"title":           "Design phase, extended",
		"end_date":        "2026-10-15",
		"seconds_per_day": float64(21600),
		"is_billable":     false,
	})

	var payload struct {
		Allocation map[string]any `json:"allocation"`
	}
	if err := json.Unmarshal(*body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %s", err)
	}
	want := map[string]any{
		"title":         "Design phase, extended",
		"endedAt":       "2026-10-15",
		"secondsPerDay": float64(21600),
		"isBillable":    false,
	}
	for key, expected := range want {
		if got := payload.Allocation[key]; got != expected {
			t.Errorf("expected %s to be %v, got %v", key, expected, got)
		}
	}
	// the identifier travels in the path, never the body
	if _, ok := payload.Allocation["id"]; ok {
		t.Error("expected id to be absent from the payload")
	}
}

// TestAllocationDeleteIsASoftDelete pins that the tool never asks for a hard
// delete. The endpoint takes hardDelete in the request body, and a hard delete
// would leave twprojects-restore_allocation with nothing to act on.
func TestAllocationDeleteIsASoftDelete(t *testing.T) {
	mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationDelete.String(), map[string]any{
		"id": float64(12345),
	})

	var payload map[string]any
	if err := json.Unmarshal(*body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %s", err)
	}
	if hardDelete, ok := payload["hardDelete"]; !ok || hardDelete != false {
		t.Errorf("expected hardDelete to be false, got %v", payload["hardDelete"])
	}
}

func TestAllocationRestore(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"allocation":{"id":12345,"status":"active"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationRestore.String(), map[string]any{
		"id": float64(12345),
	})

	if want := "/projects/api/v3/allocations/12345/restore.json"; lastURL.Path != want {
		t.Errorf("expected path %q, got %q", want, lastURL.Path)
	}
}

// TestAllocationTaskLinkPaths asserts the path each of the pair reaches. They
// take the same arguments and return the same empty body, so the path is the
// only thing that distinguishes a link from an unlink — assert it, or the two
// tools are indistinguishable to the test.
func TestAllocationTaskLinkPaths(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{{
		method: twprojects.MethodAllocationTaskLink.String(),
		want:   "/projects/api/v3/allocations/12345/link/999.json",
	}, {
		method: twprojects.MethodAllocationTaskUnlink.String(),
		want:   "/projects/api/v3/allocations/12345/unlink/999.json",
	}}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusNoContent, nil)
			testutil.ExecuteToolRequest(t, mcpServer, tt.method, map[string]any{
				"allocation_id": float64(12345),
				"task_id":       float64(999),
			})

			if lastURL.Path != tt.want {
				t.Errorf("expected path %q, got %q", tt.want, lastURL.Path)
			}
		})
	}
}

func TestAllocationGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"allocation":`+allocationBody+`}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationGet.String(), map[string]any{
		"id": float64(12345),
	})
}

// TestAllocationColorCarriesTheLeadingSign pins the one field whose spelling
// depends on which path answered. The endpoint sends six bare digits; the SDK
// models the colour as a twapi.HexColor, which restores the "#" when it
// re-encodes the typed response. So the verbose get returns "#3c8f7c" while the
// list and the sparse-fieldset get, both of which stream the body untouched,
// return "3c8f7c".
func TestAllocationColorCarriesTheLeadingSign(t *testing.T) {
	tests := []struct {
		name   string
		method string
		args   map[string]any
		body   string
		row    string
		want   string
	}{{
		name:   "verbose get re-encodes the typed colour",
		method: twprojects.MethodAllocationGet.String(),
		args:   map[string]any{"id": float64(12345)},
		body:   `{"allocation":` + allocationBody + `}`,
		row:    "allocation",
		want:   "#3c8f7c",
	}, {
		name:   "a sparse get streams the body",
		method: twprojects.MethodAllocationGet.String(),
		args:   map[string]any{"id": float64(12345), "fields": []any{"color"}},
		body:   `{"allocation":` + allocationBody + `}`,
		row:    "allocation",
		want:   "3c8f7c",
	}, {
		name:   "the list streams the body",
		method: twprojects.MethodAllocationList.String(),
		body:   `{"allocations":[` + allocationBody + `]}`,
		row:    "allocations",
		want:   "3c8f7c",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer := mcpServerMock(t, http.StatusOK, []byte(tt.body))
			testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.args,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					testutil.CheckMessage(t, result)

					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok {
						t.Fatalf("unexpected result type: %T", result)
					}
					content, ok := toolResult.Content[0].(*mcp.TextContent)
					if !ok {
						t.Fatalf("unexpected content type: %T", toolResult.Content[0])
					}

					var decoded map[string]any
					if err := json.Unmarshal([]byte(content.Text), &decoded); err != nil {
						t.Fatalf("failed to decode the response: %s", err)
					}
					row, ok := decoded[tt.row].(map[string]any)
					if !ok {
						rows, ok := decoded[tt.row].([]any)
						if !ok || len(rows) == 0 {
							t.Fatalf("no %s in the response: %s", tt.row, content.Text)
						}
						if row, ok = rows[0].(map[string]any); !ok {
							t.Fatalf("unexpected %s row: %s", tt.row, content.Text)
						}
					}
					if got := row["color"]; got != tt.want {
						t.Errorf("expected color %q but got %v", tt.want, got)
					}
				}),
			)
		})
	}
}

// TestAllocationGetSideloads pins the sideloads a verbose read asks for, and
// that financialDetails is only among them when the caller opts in: it is
// permission-gated money data, so it must not ride along by default.
func TestAllocationGetSideloads(t *testing.T) {
	tests := []struct {
		name             string
		financialDetails bool
		want             []string
	}{{
		name: "default",
		want: []string{"projects", "assignee"},
	}, {
		name:             "with financial details",
		financialDetails: true,
		want:             []string{"projects", "assignee", "financialDetails"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(`{"allocation":`+allocationBody+`}`))
			args := map[string]any{"id": float64(12345)}
			if tt.financialDetails {
				args["include_financial_details"] = true
			}
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationGet.String(), args)

			// One comma-separated parameter, not one parameter per sideload:
			// the endpoint reads only the first `include` it receives and drops
			// repeated ones, so the request would succeed carrying just the
			// first sideload. Assert the raw query, because url.Values renders
			// both encodings as a slice and hides the difference.
			raw := lastURL.RawQuery
			if got := strings.Count(raw, "include="); got != 1 {
				t.Errorf("expected exactly one include parameter, got %d in %q", got, raw)
			}
			if want := strings.Join(tt.want, ","); lastURL.Query().Get("include") != want {
				t.Errorf("expected include=%q, got %q", want, lastURL.Query().Get("include"))
			}
		})
	}
}

// TestAllocationGetSparseFieldsDropsSideloads pins that a field selection wins
// over the sideloads: sideloading would hand back the bulk the selection exists
// to avoid.
func TestAllocationGetSparseFieldsDropsSideloads(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"allocation":`+allocationBody+`}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationGet.String(), map[string]any{
		"id":     float64(12345),
		"fields": []any{"title"},
	})

	if includes := lastURL.Query()["include"]; len(includes) > 0 {
		t.Errorf("expected no sideloads alongside a field selection, got %v", includes)
	}
	if got := lastURL.Query().Get("fields[allocations]"); got != "title,id" && got != "id,title" {
		t.Errorf("expected the selection plus id, got %q", got)
	}
}

func TestAllocationList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"allocations":[`+allocationBody+`]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), map[string]any{
		"start_date": "2026-09-01",
		"end_date":   "2026-09-30",
		"page":       float64(1),
		"page_size":  float64(10),
	})
}

// TestAllocationListFiltersReachTheWire drives every filter the tool advertises
// against the query string. The mock replies with the same body either way, so a
// filter that never reaches the request looks identical to a working one.
func TestAllocationListFiltersReachTheWire(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"allocations":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), map[string]any{
		"start_date":             "2026-09-01",
		"end_date":               "2026-09-30",
		"search_term":            "design",
		"assigned_user_ids":      []float64{1, 2},
		"assigned_user_team_ids": []float64{3},
		"project_ids":            []float64{777},
		"project_owner_ids":      []float64{4},
		"project_category_ids":   []float64{5},
		"project_company_ids":    []float64{6},
		"project_tag_ids":        []float64{7, 8},
		"match_all_project_tags": true,
		"project_status":         "active",
		"updated_after":          "2026-08-01T09:00:00Z",
		"deleted_after":          "2026-08-02",
		"show_deleted":           true,
		"page":                   float64(2),
		"page_size":              float64(25),
	})

	query := lastURL.Query()
	want := map[string]string{
		"startDate":           "2026-09-01",
		"endDate":             "2026-09-30",
		"searchTerm":          "design",
		"assignedUserIds":     "1,2",
		"assignedUserTeamIds": "3",
		"projectIds":          "777",
		"projectOwnerIds":     "4",
		"projectCategoryIds":  "5",
		"projectCompanyIds":   "6",
		"projectTagIds":       "7,8",
		"matchAllProjectTags": "true",
		"projectStatus":       "active",
		"showDeleted":         "true",
		"page":                "2",
		"pageSize":            "25",
	}
	for key, expected := range want {
		if got := query.Get(key); got != expected {
			t.Errorf("expected %s to be %q, got %q", key, expected, got)
		}
	}
	if got := query.Get("updatedAfter"); got != "2026-08-01T09:00:00Z" {
		t.Errorf("expected updatedAfter to be an RFC 3339 timestamp, got %q", got)
	}
	// a plain YYYY-MM-DD is accepted on a date-time filter and widened to a
	// timestamp, because models emit the bare date by default
	if got := query.Get("deletedAfter"); got == "" || got == "2026-08-02" {
		t.Errorf("expected deletedAfter to be normalised to a timestamp, got %q", got)
	}
}

// TestAllocationListWindowIsNotDefaultedLocally pins that omitting the dates
// sends no window at all, leaving the endpoint's own default in place. The
// endpoint answers an unbounded call with today through thirty days from today
// and says nothing about having narrowed the range, which is why the tool
// description tells callers to pass both — but inventing a window here would
// silently reorder what every existing caller gets back.
func TestAllocationListWindowIsNotDefaultedLocally(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"allocations":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), map[string]any{})

	for _, key := range []string{"startDate", "endDate"} {
		if got := lastURL.Query().Get(key); got != "" {
			t.Errorf("expected %s to be absent, got %q", key, got)
		}
	}
}

// TestAllocationListVerboseFalseAsksForFewFields pins the minimal selection, and
// that it drops the sideloads a verbose read would request.
func TestAllocationListVerboseFalseAsksForFewFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"allocations":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), map[string]any{
		"verbose": false,
	})

	if includes := lastURL.Query()["include"]; len(includes) > 0 {
		t.Errorf("expected no sideloads when verbose is false, got %v", includes)
	}
	if got := lastURL.Query().Get("fields[allocations]"); got == "" {
		t.Error("expected a minimal field selection when verbose is false")
	}
}

// TestAllocationListFinancialDetailsIsOptIn pins that the gated money sideload
// is only requested when the caller asks for it.
func TestAllocationListFinancialDetailsIsOptIn(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want bool
	}{
		{name: "default", args: map[string]any{}, want: false},
		{name: "opted in", args: map[string]any{"include_financial_details": true}, want: true},
		{name: "opted out", args: map[string]any{"include_financial_details": false}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(`{"allocations":[]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), tt.args)

			// the sideloads travel as one comma-separated value, so this is a
			// membership check within it rather than over separate parameters
			found := slices.Contains(
				strings.Split(lastURL.Query().Get("include"), ","),
				"financialDetails",
			)
			if found != tt.want {
				t.Errorf("expected financialDetails sideload %v, got %v", tt.want, found)
			}
		})
	}
}

// TestAllocationValidatesAgainstOutputSchema validates a response carrying real
// dates against the schema the tools publish. twapi.Date and time.Time are
// described by reflection, and without helpers.WithDateTypeSchema they reflect
// as opaque objects while their MarshalJSON writes strings — so every row would
// fail validation at a validating client while passing every other test here,
// since neither the server nor the harness validates structured content.
func TestAllocationValidatesAgainstOutputSchema(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"allocation":`+allocationBody+`}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationGet.String(), map[string]any{
		"id": float64(12345),
	}, testutil.ExecuteToolRequestWithCheckMessage(
		checkStructuredContentMatchesOutputSchema(twprojects.MethodAllocationGet.String()),
	))

	mcpServer = mcpServerMock(t, http.StatusOK, []byte(`{"allocations":[`+allocationBody+`]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationList.String(), map[string]any{
		"start_date": "2026-09-01",
		"end_date":   "2026-09-30",
	}, testutil.ExecuteToolRequestWithCheckMessage(
		checkStructuredContentMatchesOutputSchema(twprojects.MethodAllocationList.String()),
	))
}

// TestAllocationAPIFailuresAreToolResults pins that a status the caller can act
// on comes back as an error tool result rather than a Go error. A raw error
// becomes a protocol-level failure, which hands the model a transport error with
// no status to read and none of the handler's wrapped message.
func TestAllocationAPIFailuresAreToolResults(t *testing.T) {
	methods := []struct {
		method string
		args   map[string]any
	}{{
		method: twprojects.MethodAllocationGet.String(),
		args:   map[string]any{"id": float64(12345)},
	}, {
		method: twprojects.MethodAllocationList.String(),
		args:   map[string]any{},
	}, {
		method: twprojects.MethodAllocationCreate.String(),
		args: map[string]any{
			"project_id":       float64(777),
			"assigned_user_id": float64(456),
			"title":            "Design phase",
			"start_date":       "2026-09-01",
			"end_date":         "2026-09-30",
			"seconds_per_day":  float64(14400),
			"color":            "#3c8f7c",
		},
	}, {
		method: twprojects.MethodAllocationUpdate.String(),
		args:   map[string]any{"id": float64(12345), "title": "Renamed"},
	}, {
		method: twprojects.MethodAllocationDelete.String(),
		args:   map[string]any{"id": float64(12345)},
	}, {
		method: twprojects.MethodAllocationRestore.String(),
		args:   map[string]any{"id": float64(12345)},
	}, {
		method: twprojects.MethodAllocationTaskLink.String(),
		args:   map[string]any{"allocation_id": float64(12345), "task_id": float64(999)},
	}, {
		method: twprojects.MethodAllocationTaskUnlink.String(),
		args:   map[string]any{"allocation_id": float64(12345), "task_id": float64(999)},
	}}

	statuses := []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError}

	for _, tt := range methods {
		for _, status := range statuses {
			t.Run(tt.method, func(t *testing.T) {
				mcpServer := mcpServerMock(t, status, []byte(`{"errors":[{"title":"nope"}]}`))
				testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.args,
					testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
						t.Helper()
						toolResult, ok := result.(*mcp.CallToolResult)
						if !ok {
							t.Fatalf("unexpected result type: %T", result)
						}
						if !toolResult.IsError {
							t.Errorf("expected %s to report status %d as an error tool result", tt.method, status)
						}
					}))
			})
		}
	}
}

// TestAllocationUnmodelledAttributesAreNotWritable pins the invariant behind
// leaving distributeType and linkedTaskLoggedTime unmodelled rather than
// stripping them from responses: neither is exposed as a parameter, so no
// caller can set one. list_allocations streams the API's body verbatim and
// therefore still returns both, which is accepted precisely because they are
// read-only from here — an argument naming either is dropped by the binder and
// never reaches the payload, whichever convention it is spelled in.
func TestAllocationUnmodelledAttributesAreNotWritable(t *testing.T) {
	required := map[string]any{
		"project_id":       float64(777),
		"assigned_user_id": float64(456),
		"title":            "Design phase",
		"start_date":       "2026-09-01",
		"end_date":         "2026-09-30",
		"seconds_per_day":  float64(14400),
		"color":            "#3c8f7c",
	}

	for _, key := range []string{
		"distribute_type", "distributeType",
		"linked_task_logged_time", "linkedTaskLoggedTime",
	} {
		t.Run(key, func(t *testing.T) {
			args := maps.Clone(required)
			args[key] = "distributed"

			mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusCreated,
				[]byte(`{"allocation":{"id":12345}}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreate.String(), args)

			sent := string(*body)
			for _, unwritable := range []string{"distributeType", "linkedTaskLoggedTime"} {
				if strings.Contains(sent, unwritable) {
					t.Errorf("argument %q put %s on the wire: %s", key, unwritable, sent)
				}
			}
		})
	}
}

// TestAllocationOverAllocationIsReportedNotHidden pins the two halves of the
// capacity posture. informOfOverAllocation is defaulted on, so a change that
// overruns the user's capacity goes through instead of being refused; and when
// the API reports it, the result says so rather than reading as a clean
// success. Without the second half the flag would be worse than useless — the
// change would land and nobody would be told.
func TestAllocationOverAllocationIsReportedNotHidden(t *testing.T) {
	required := map[string]any{
		"project_id":       float64(777),
		"assigned_user_id": float64(456),
		"title":            "Design phase",
		"start_date":       "2026-09-01",
		"end_date":         "2026-09-30",
		"seconds_per_day":  float64(14400),
		"color":            "#3c8f7c",
	}

	t.Run("defaulted on when unset", func(t *testing.T) {
		mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusCreated,
			[]byte(`{"allocation":{"id":12345}}`))
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreate.String(), required)

		var payload struct {
			Allocation map[string]any `json:"allocation"`
		}
		if err := json.Unmarshal(*body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		if payload.Allocation["informOfOverAllocation"] != true {
			t.Errorf("expected informOfOverAllocation to default to true, got %v",
				payload.Allocation["informOfOverAllocation"])
		}
	})

	t.Run("caller can turn it off", func(t *testing.T) {
		args := maps.Clone(required)
		args["inform_of_over_allocation"] = false

		mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusCreated,
			[]byte(`{"allocation":{"id":12345}}`))
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationCreate.String(), args)

		var payload struct {
			Allocation map[string]any `json:"allocation"`
		}
		if err := json.Unmarshal(*body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		if payload.Allocation["informOfOverAllocation"] != false {
			t.Errorf("expected an explicit false to survive, got %v",
				payload.Allocation["informOfOverAllocation"])
		}
	})

	for _, tc := range []struct {
		name   string
		method string
		status int
		args   map[string]any
	}{{
		name:   "create",
		method: twprojects.MethodAllocationCreate.String(),
		status: http.StatusCreated,
		args:   required,
	}, {
		name:   "update",
		method: twprojects.MethodAllocationUpdate.String(),
		status: http.StatusOK,
		args:   map[string]any{"id": float64(12345), "end_date": "2026-10-31"},
	}} {
		t.Run(tc.name+" reports it back", func(t *testing.T) {
			mcpServer := mcpServerMock(t, tc.status,
				[]byte(`{"allocation":{"id":12345,"overAllocated":true}}`))
			testutil.ExecuteToolRequest(t, mcpServer, tc.method, tc.args,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					testutil.CheckMessage(t, result)

					var text string
					for _, content := range result.(*mcp.CallToolResult).Content {
						if textContent, ok := content.(*mcp.TextContent); ok {
							text += textContent.Text
						}
					}
					if !strings.Contains(text, "over their capacity") {
						t.Errorf("expected the result to report the over-allocation, got %q", text)
					}
				}))
		})
	}
}

// TestAllocationGetReturnsSideloads pins that the sideloads the tool requests
// reach the caller. The typed response carried no Included struct at first, so
// get_allocation asked for include=projects,assignee and then dropped both on
// decode — the query string looked correct and the related objects never
// arrived. The result is also validated against the published output schema,
// since the sideloads widened it.
func TestAllocationGetReturnsSideloads(t *testing.T) {
	body := `{"allocation":` + allocationBody + `,"included":{
		"projects":{"777":{"id":777,"name":"Example Project"}},
		"users":{"456":{"id":456,"firstName":"John","lastName":"Doe"}},
		"jobRoles":{"222":{"id":222,"name":"Creative Director"}}
	}}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(body))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodAllocationGet.String(), map[string]any{
		"id": float64(12345),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		checkStructuredContentMatchesOutputSchema(twprojects.MethodAllocationGet.String())(t, result)

		var text string
		for _, content := range result.(*mcp.CallToolResult).Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				text += textContent.Text
			}
		}
		for _, want := range []string{"Example Project", "John", "Creative Director"} {
			if !strings.Contains(text, want) {
				t.Errorf("expected the sideloaded %q to reach the caller, got %s", want, text)
			}
		}
	}))
}
