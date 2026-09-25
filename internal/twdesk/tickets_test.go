//nolint:lll
package twdesk_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/pkg/toolsets"
)

func TestTicketCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket":{"id":123,"subject":"Test Ticket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketCreate.String(), map[string]any{
		"subject":        "Test Ticket",
		"body":           "This is a test ticket",
		"inboxId":        float64(1),
		"notifyCustomer": nil,
		"cc":             []string{"cc@example.com"},
		"bcc":            []string{"bcc@example.com"},
		"files":          nil,
		"tags":           nil,
		"priorityId":     float64(1),
		"statusId":       float64(1),
		"typeId":         float64(1),
		"customerId":     float64(100),
		"customerEmail":  nil,
		"agentId":        float64(1),
	})
}

func TestTicketUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Updated Ticket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketUpdate.String(), map[string]any{
		"id":         float64(123),
		"subject":    "Updated Ticket",
		"body":       nil,
		"tags":       nil,
		"deleteTags": nil,
		"cc":         []string{"cc-update@example.com"},
		"bcc":        []string{"bcc-update@example.com"},
		"inboxId":    nil,
		"priorityId": float64(2),
		"statusId":   float64(2),
		"typeId":     float64(2),
		"agentId":    nil,
	})
}

func TestTicketGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Test Ticket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketGet.String(), map[string]any{
		"id":     float64(123),
		"fields": nil,
	})
}

func TestTicketSearch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"},{"id":124,"subject":"Ticket 2"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), map[string]any{
		"search":         "Testing 123",
		"inboxIDs":       nil,
		"customerIDs":    nil,
		"companyIDs":     nil,
		"tagIDs":         nil,
		"statusIDs":      []float64{1, 2},
		"priorityIDs":    []float64{1, 2, 3},
		"userIDs":        nil,
		"createdAfter":   nil,
		"createdBefore":  nil,
		"omitMerged":     nil,
		"page":           float64(1),
		"pageSize":       float64(10),
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

// TestTicketSearchForwardsPaginationAndFields pins the query string the tool
// builds. The tool documents page, pageSize, orderBy, orderDirection and fields,
// but the SDK's Tickets.Search encodes only the filter struct, so all five were
// silently dropped and every call returned the endpoint's default first page
// with the full attribute set.
func TestTicketSearchForwardsPaginationAndFields(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
		http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), map[string]any{
		"search":         "Testing 123",
		"inboxIDs":       nil,
		"customerIDs":    nil,
		"companyIDs":     nil,
		"tagIDs":         nil,
		"statusIDs":      []float64{1, 2},
		"priorityIDs":    nil,
		"userIDs":        nil,
		"createdAfter":   nil,
		"createdBefore":  nil,
		"omitMerged":     nil,
		"page":           float64(3),
		"pageSize":       float64(200),
		"orderBy":        "createdAt",
		"orderDirection": "asc",
		"fields":         []string{"id", "subject"},
	})

	requestURL := lastRequestURL()
	if got, want := requestURL.Path, "/desk/api/v2/search/tickets.json"; got != want {
		t.Errorf("unexpected request path: got %q, want %q", got, want)
	}

	query := requestURL.Query()
	for key, want := range map[string]string{
		"page":      "3",
		"pageSize":  "200",
		"orderBy":   "createdAt",
		"orderMode": "asc",
		"fields":    "id,subject",
		"search":    "Testing 123",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("query parameter %q: got %q, want %q", key, got, want)
		}
	}

	// The filter itself still goes over the wire exactly as the SDK's
	// Tickets.Search encoded it: one repeated parameter per ID, not a
	// comma-joined list.
	if got, want := query["statuses"], []string{"1", "2"}; !slices.Equal(got, want) {
		t.Errorf("query parameter \"statuses\": got %v, want %v", got, want)
	}
}

// TestTicketSearchDefaultsPaginationWithoutOrdering checks the defaults applied
// when the caller supplies no pagination. Ordering is deliberately absent: the
// search endpoint documents no ordering parameter, so it is only forwarded when
// explicitly requested.
func TestTicketSearchDefaultsPaginationWithoutOrdering(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
		http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), map[string]any{
		"search":         "Testing 123",
		"inboxIDs":       nil,
		"customerIDs":    nil,
		"companyIDs":     nil,
		"tagIDs":         nil,
		"statusIDs":      nil,
		"priorityIDs":    nil,
		"userIDs":        nil,
		"createdAfter":   nil,
		"createdBefore":  nil,
		"omitMerged":     nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})

	requestURL := lastRequestURL()
	query := requestURL.Query()
	for key, want := range map[string]string{
		"page":     "1",
		"pageSize": "10",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("query parameter %q: got %q, want %q", key, got, want)
		}
	}
	for _, key := range []string{"orderBy", "orderMode", "fields", "startDate", "endDate"} {
		if query.Has(key) {
			t.Errorf("query parameter %q should not be sent when not requested, got %q", key, query.Get(key))
		}
	}
}

// TestTicketSearchForwardsCreatedDateRange pins the creation-date window onto
// the query string as the endpoint's startDate/endDate, in the plain YYYY-MM-DD
// form the Desk web app's own search sends. An RFC 3339 value is truncated to
// its day rather than forwarded: qs would render the filter struct's *time.Time
// fields as RFC 3339, and sending that shape is what this tool first shipped
// with and what the endpoint rejected.
func TestTicketSearchForwardsCreatedDateRange(t *testing.T) {
	tests := []struct {
		name                           string
		createdAfter                   any
		createdBefore                  any
		wantStart, wantEnd             string
		wantStartAbsent, wantEndAbsent bool
	}{{
		name:          "plain dates",
		createdAfter:  "2026-08-05",
		createdBefore: "2026-08-06",
		wantStart:     "2026-08-05",
		wantEnd:       "2026-08-06",
	}, {
		name:          "timestamps truncated to the day",
		createdAfter:  "2026-08-05T14:30:00Z",
		createdBefore: "2026-08-06T09:15:00Z",
		wantStart:     "2026-08-05",
		wantEnd:       "2026-08-06",
	}, {
		name:            "open-ended lower bound",
		createdAfter:    nil,
		createdBefore:   "2026-08-06",
		wantStartAbsent: true,
		wantEnd:         "2026-08-06",
	}, {
		name:          "open-ended upper bound",
		createdAfter:  "2026-08-05",
		createdBefore: nil,
		wantStart:     "2026-08-05",
		wantEndAbsent: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
				http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), map[string]any{
				"search":         "Testing 123",
				"inboxIDs":       nil,
				"customerIDs":    nil,
				"companyIDs":     nil,
				"tagIDs":         nil,
				"statusIDs":      nil,
				"priorityIDs":    nil,
				"userIDs":        nil,
				"createdAfter":   tt.createdAfter,
				"createdBefore":  tt.createdBefore,
				"omitMerged":     nil,
				"page":           nil,
				"pageSize":       nil,
				"orderBy":        nil,
				"orderDirection": nil,
				"fields":         nil,
			})

			requestURL := lastRequestURL()
			query := requestURL.Query()
			for _, bound := range []struct {
				key    string
				want   string
				absent bool
			}{
				{"startDate", tt.wantStart, tt.wantStartAbsent},
				{"endDate", tt.wantEnd, tt.wantEndAbsent},
			} {
				switch {
				case bound.absent:
					if query.Has(bound.key) {
						t.Errorf("query parameter %q should be absent, got %q", bound.key, query.Get(bound.key))
					}
				default:
					if got := query.Get(bound.key); got != bound.want {
						t.Errorf("query parameter %q: got %q, want %q", bound.key, got, bound.want)
					}
				}
			}
		})
	}
}

// TestTicketSearchRejectsInvalidCreatedDate keeps an unparseable date an
// explicit error rather than a silently dropped filter, which would look like a
// wider result set than the caller asked for.
func TestTicketSearchRejectsInvalidCreatedDate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), map[string]any{
		"search":         "Testing 123",
		"inboxIDs":       nil,
		"customerIDs":    nil,
		"companyIDs":     nil,
		"tagIDs":         nil,
		"statusIDs":      nil,
		"priorityIDs":    nil,
		"userIDs":        nil,
		"createdAfter":   "last tuesday",
		"createdBefore":  nil,
		"omitMerged":     nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Fatal("an unparseable createdAfter should be an error, not a silently dropped filter")
		}
		textContent, ok := toolResult.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type: %T", toolResult.Content[0])
		}
		if !strings.Contains(textContent.Text, "createdAfter") {
			t.Errorf("error should name the offending parameter, got %q", textContent.Text)
		}
	}))
}

// TestTicketSearchFiltersReachTheWire pins every filter beyond the original
// set onto the query string under the name the endpoint reads. The mock answers
// the same body whatever is sent, so a filter dropped on the way would pass any
// check on the result.
func TestTicketSearchFiltersReachTheWire(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
		http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), searchArgs(map[string]any{
		"tagIDs":                 []float64{1},
		"excludeTagIDs":          []float64{2, 3},
		"requireAllTags":         true,
		"typeIDs":                []float64{4},
		"sourceIDs":              []float64{5, 6},
		"happinessRatingIDs":     []float64{7, 8},
		"includeArchivedAgents":  true,
		"subjectKeywords":        []string{"invoice", "refund"},
		"excludePersonalInboxes": true,
		"teamworkCompanyIDs":     []float64{9},
		"taskStatuses":           []string{"active", "complete"},
		"onlyWithAttachments":    true,
		"taskID":                 float64(10),
		"projectID":              float64(11),
		"exact":                  true,
	}))

	requestURL := lastRequestURL()
	query := requestURL.Query()
	for key, want := range map[string][]string{
		"tags":                  {"1"},
		"excludeTags":           {"2", "3"},
		"tagRequireAll":         {"true"},
		"types":                 {"4"},
		"sources":               {"5", "6"},
		"rating":                {"7", "8"},
		"includeArchivedAgents": {"true"},
		"subjectKeywords":       {"invoice", "refund"},
		"excludeWorkEmails":     {"true"},
		"twCompanyIds":          {"9"},
		"taskStatuses":          {"active", "complete"},
		"onlyWithAttachment":    {"true"},
		"task":                  {"10"},
		"project":               {"11"},
		"exact":                 {"true"},
	} {
		if got := query[key]; !slices.Equal(got, want) {
			t.Errorf("query parameter %q: got %v, want %v", key, got, want)
		}
	}
}

// TestTicketSearchBooleanFiltersReachTheWire covers the two filters that
// cannot share a request with the ones above: each rejects the other half of a
// pair that the endpoint would answer with an empty or silently widened list.
func TestTicketSearchBooleanFiltersReachTheWire(t *testing.T) {
	for _, key := range []struct{ arg, param string }{
		{"unassigned", "unassigned"},
		{"onlyUntagged", "onlyUntagged"},
	} {
		t.Run(key.arg, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
				http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(),
				searchArgs(map[string]any{key.arg: true}))

			requestURL := lastRequestURL()
			if got := requestURL.Query().Get(key.param); got != "true" {
				t.Errorf("query parameter %q: got %q, want %q", key.param, got, "true")
			}
		})
	}
}

// TestTicketSearchForwardsUpdatedDateRange pins the last-update window. Unlike
// the creation dates, the endpoint reads these as RFC 3339 and compares them as
// sent, so a plain date must become the day's first and last second rather than
// being forwarded as a date.
func TestTicketSearchForwardsUpdatedDateRange(t *testing.T) {
	tests := []struct {
		name                      string
		updatedAfter              any
		updatedBefore             any
		wantFrom, wantTo          string
		wantFromAbsent, wantToAbs bool
	}{{
		name:          "plain dates cover whole days",
		updatedAfter:  "2026-08-01",
		updatedBefore: "2026-08-31",
		wantFrom:      "2026-08-01T00:00:00Z",
		wantTo:        "2026-08-31T23:59:59Z",
	}, {
		name:          "timestamps are kept and normalised to UTC",
		updatedAfter:  "2026-08-01T09:30:00+02:00",
		updatedBefore: "2026-08-31T17:00:00Z",
		wantFrom:      "2026-08-01T07:30:00Z",
		wantTo:        "2026-08-31T17:00:00Z",
	}, {
		name:           "open-ended lower bound",
		updatedBefore:  "2026-08-31",
		wantFromAbsent: true,
		wantTo:         "2026-08-31T23:59:59Z",
	}, {
		name:         "open-ended upper bound",
		updatedAfter: "2026-08-01",
		wantFrom:     "2026-08-01T00:00:00Z",
		wantToAbs:    true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
				http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), searchArgs(map[string]any{
				"updatedAfter":  tt.updatedAfter,
				"updatedBefore": tt.updatedBefore,
			}))

			requestURL := lastRequestURL()
			query := requestURL.Query()
			for _, bound := range []struct {
				key    string
				want   string
				absent bool
			}{
				{"updatedAtFrom", tt.wantFrom, tt.wantFromAbsent},
				{"updatedAtTo", tt.wantTo, tt.wantToAbs},
			} {
				switch {
				case bound.absent:
					if query.Has(bound.key) {
						t.Errorf("query parameter %q should be absent, got %q", bound.key, query.Get(bound.key))
					}
				default:
					if got := query.Get(bound.key); got != bound.want {
						t.Errorf("query parameter %q: got %q, want %q", bound.key, got, bound.want)
					}
				}
			}
			// The update window must not leak into the creation window.
			for _, key := range []string{"startDate", "endDate", "lastUpdated"} {
				if query.Has(key) {
					t.Errorf("query parameter %q should be absent, got %q", key, query.Get(key))
				}
			}
		})
	}
}

// TestTicketSearchForwardsCustomFields pins the custom-field conditions as the
// single JSON document the endpoint reads from customfields.
func TestTicketSearchForwardsCustomFields(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
		http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), searchArgs(map[string]any{
		"customFields": []any{
			map[string]any{"id": float64(12), "value": "gold", "operation": "contains"},
			map[string]any{"id": float64(13), "values": []any{float64(14), float64(15)}},
		},
	}))

	want := `[{"id":12,"value":"gold","operation":"contains"},{"id":13,"values":[14,15]}]`
	requestURL := lastRequestURL()
	if got := requestURL.Query()["customfields"]; !slices.Equal(got, []string{want}) {
		t.Errorf("query parameter \"customfields\": got %v, want [%s]", got, want)
	}
}

// TestTicketSearchOmittedFiltersSendNothing keeps the parameters the handler
// writes by hand off the query string when the caller did not ask for them.
func TestTicketSearchOmittedFiltersSendNothing(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
		http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), searchArgs(nil))

	requestURL := lastRequestURL()
	query := requestURL.Query()
	for _, key := range []string{"updatedAtFrom", "updatedAtTo", "rating", "customfields", "project"} {
		if query.Has(key) {
			t.Errorf("query parameter %q should not be sent when not requested, got %v", key, query[key])
		}
	}
}

// TestTicketSearchRejectsInvalidFilters keeps each combination the endpoint
// would answer with an empty or silently widened list an explicit error.
func TestTicketSearchRejectsInvalidFilters(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		wantText string
	}{{
		name:     "unassigned with userIDs",
		args:     map[string]any{"unassigned": true, "userIDs": []float64{1}},
		wantText: "unassigned",
	}, {
		name:     "onlyUntagged with tagIDs",
		args:     map[string]any{"onlyUntagged": true, "tagIDs": []float64{1}},
		wantText: "onlyUntagged",
	}, {
		name:     "onlyUntagged with excludeTagIDs",
		args:     map[string]any{"onlyUntagged": true, "excludeTagIDs": []float64{1}},
		wantText: "onlyUntagged",
	}, {
		name:     "unknown task status",
		args:     map[string]any{"taskStatuses": []string{"late"}},
		wantText: "taskStatuses",
	}, {
		name:     "custom field without a value",
		args:     map[string]any{"customFields": []any{map[string]any{"id": float64(12)}}},
		wantText: "customFields[0]",
	}, {
		name: "custom field with an unknown operation",
		args: map[string]any{"customFields": []any{
			map[string]any{"id": float64(12), "value": "x", "operation": "like"},
		}},
		wantText: "operation",
	}, {
		name:     "unparseable updated date",
		args:     map[string]any{"updatedAfter": "last month"},
		wantText: "updatedAfter",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), searchArgs(tt.args),
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()

					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok {
						t.Fatalf("unexpected result type: %T", result)
					}
					if !toolResult.IsError {
						t.Fatal("expected an error tool result")
					}
					textContent, ok := toolResult.Content[0].(*mcp.TextContent)
					if !ok {
						t.Fatalf("unexpected content type: %T", toolResult.Content[0])
					}
					if !strings.Contains(textContent.Text, tt.wantText) {
						t.Errorf("error should name %q, got %q", tt.wantText, textContent.Text)
					}
				}))
		})
	}
}

func TestTicketTaskLink(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, nil)
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketTaskLink.String(), map[string]any{
		"ticketId": float64(123),
		"taskId":   float64(456),
	})
}

func TestTicketTaskUnlink(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, nil)
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketTaskUnlink.String(), map[string]any{
		"ticketId": float64(123),
		"taskId":   float64(456),
	})
}

// TestTicketTaskLinkRequests pins the method and path each tool sends. The two
// tools address the same endpoint and differ only in POST versus DELETE, so a
// check on the URL alone would pass with the two swapped.
func TestTicketTaskLinkRequests(t *testing.T) {
	tests := []struct {
		name       string
		method     toolsets.Method
		wantMethod string
	}{
		{name: "link", method: twdesk.MethodTicketTaskLink, wantMethod: http.MethodPost},
		{name: "unlink", method: twdesk.MethodTicketTaskUnlink, wantMethod: http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequest, cleanup := testutil.DeskMCPServerMockWithRequest(t, http.StatusOK, nil)
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, tt.method.String(), map[string]any{
				"ticketId": float64(123),
				"taskId":   float64(456),
			})

			method, requestURL := lastRequest()
			if method != tt.wantMethod {
				t.Errorf("unexpected request method: got %q, want %q", method, tt.wantMethod)
			}
			if got, want := requestURL.Path, "/desk/api/v2/tickets/123/tasks/456.json"; got != want {
				t.Errorf("unexpected request path: got %q, want %q", got, want)
			}
		})
	}
}

// TestTicketTaskLinkRejectsNonPositiveIDs keeps bad caller input a tool result.
// The SDK rejects a non-positive ID with a bare Go error carrying no status,
// which HandleAPIError can only hand back as a protocol-level failure.
func TestTicketTaskLinkRejectsNonPositiveIDs(t *testing.T) {
	tests := []struct {
		name     string
		method   toolsets.Method
		args     map[string]any
		wantText string
	}{
		{
			name:     "link/ticket",
			method:   twdesk.MethodTicketTaskLink,
			args:     map[string]any{"ticketId": float64(0), "taskId": float64(456)},
			wantText: "ticketId",
		},
		{
			name:     "link/task",
			method:   twdesk.MethodTicketTaskLink,
			args:     map[string]any{"ticketId": float64(123), "taskId": float64(-1)},
			wantText: "taskId",
		},
		{
			name:     "unlink/ticket",
			method:   twdesk.MethodTicketTaskUnlink,
			args:     map[string]any{"ticketId": float64(-1), "taskId": float64(456)},
			wantText: "ticketId",
		},
		{
			name:     "unlink/task",
			method:   twdesk.MethodTicketTaskUnlink,
			args:     map[string]any{"ticketId": float64(123), "taskId": float64(0)},
			wantText: "taskId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, cleanup := mcpServerMock(t, http.StatusOK, nil)
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, tt.method.String(), tt.args,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()

					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok {
						t.Fatalf("unexpected result type: %T", result)
					}
					if !toolResult.IsError {
						t.Fatal("a non-positive ID should be an error tool result")
					}
					textContent, ok := toolResult.Content[0].(*mcp.TextContent)
					if !ok {
						t.Fatalf("unexpected content type: %T", toolResult.Content[0])
					}
					if !strings.Contains(textContent.Text, tt.wantText) {
						t.Errorf("error should name the offending parameter, got %q", textContent.Text)
					}
				}))
		})
	}
}

// searchArgs returns a fully populated twdesk-search_tickets argument map, so a
// test can override only the parameter it is about. Every parameter is required
// by the strict-mode schema, so none may be left out.
func searchArgs(overrides map[string]any) map[string]any {
	args := map[string]any{
		"search": nil, "inboxIDs": nil, "customerIDs": nil, "companyIDs": nil,
		"tagIDs": nil, "statusIDs": nil, "priorityIDs": nil, "userIDs": nil,
		"createdAfter": nil, "createdBefore": nil, "omitMerged": nil,
		"updatedAfter": nil, "updatedBefore": nil, "excludeTagIDs": nil,
		"requireAllTags": nil, "onlyUntagged": nil, "typeIDs": nil, "sourceIDs": nil,
		"happinessRatingIDs": nil, "unassigned": nil, "includeArchivedAgents": nil,
		"subjectKeywords": nil, "excludePersonalInboxes": nil, "teamworkCompanyIDs": nil,
		"taskStatuses": nil, "onlyWithAttachments": nil, "taskID": nil, "projectID": nil,
		"exact": nil, "customFields": nil,
		"page": nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil,
		"fields": nil,
	}
	for key, value := range overrides {
		args[key] = value
	}
	return args
}

// searchResultText runs a search and hands the tool's text content to check.
func searchResultText(t *testing.T, mcpServer *mcp.Server, args map[string]any, check func(*testing.T, string)) {
	t.Helper()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(), args,
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()

			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if len(toolResult.Content) == 0 {
				t.Fatal("tool result should carry content")
			}
			textContent, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("unexpected content type: %T", toolResult.Content[0])
			}
			check(t, textContent.Text)
		}),
	)
}

// TestTicketSearchOmitMergedReachesTheWire pins the merged-ticket exclusion on
// the query string. Merged is not a ticket status, so statusIDs cannot express
// it and a caller asked to leave merged tickets out has no other parameter to
// reach for. The mock answers the same body either way.
func TestTicketSearchOmitMergedReachesTheWire(t *testing.T) {
	for _, tt := range []struct {
		name       string
		omitMerged any
		want       string
	}{
		{name: "requested", omitMerged: true, want: "true"},
		{name: "declined", omitMerged: false, want: "false"},
		{name: "omitted", omitMerged: nil, want: "false"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
				http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(),
				searchArgs(map[string]any{"omitMerged": tt.omitMerged}))

			requestURL := lastRequestURL()
			if got := requestURL.Query().Get("omitMerged"); got != tt.want {
				t.Errorf("query parameter \"omitMerged\": got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTicketSearchAppliesFieldsToTheResponse pins the sparse fieldset being
// applied here rather than by the endpoint. /search/tickets.json ignores the
// fields query parameter and answers with the whole record — including one
// reference object per activity, message, file and timelog the ticket has —
// so a page of them is large enough to be truncated before it reaches the
// caller, which is indistinguishable from a page that returned nothing.
func TestTicketSearchAppliesFieldsToTheResponse(t *testing.T) {
	// The body the endpoint answers with whatever fields was asked for.
	body := []byte(`{"tickets":[{"id":123,"subject":"Ticket 1","previewText":"a long preview",` +
		`"activities":[{"id":1},{"id":2}],"messages":[{"id":3}],"createdAt":"2026-06-01T00:00:00Z"}],` +
		`"included":{"messages":null},"pagination":{"records":10000,"pageSize":10,"pages":10000,` +
		`"page":1,"hasMorePages":true}}`)

	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, body)
	defer cleanup()

	searchResultText(t, mcpServer, searchArgs(map[string]any{
		"fields": []string{"subject", "createdAt"},
	}), func(t *testing.T, text string) {
		t.Helper()

		for _, want := range []string{`"subject"`, `"createdAt"`, `"id"`} {
			if !strings.Contains(text, want) {
				t.Errorf("selected attribute %s should be returned, got %q", want, text)
			}
		}
		// id is always kept so a row stays addressable by twdesk-get_ticket.
		for _, unwanted := range []string{"activities", "messages", "previewText", "included"} {
			if strings.Contains(text, unwanted) {
				t.Errorf("unselected attribute %q should be dropped, got %q", unwanted, text)
			}
		}
	})
}

// TestTicketSearchRejectsAnUnknownField keeps the published vocabulary and the
// handler in step: a name the schema does not list is refused rather than
// silently returning a row with nothing but its id.
func TestTicketSearchRejectsAnUnknownField(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(),
		searchArgs(map[string]any{"fields": []string{"id", "nosuchattribute"}}),
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()

			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if !toolResult.IsError {
				t.Fatal("an unknown ticket attribute should be an error, not a silently dropped selection")
			}
		}),
	)
}

// TestTicketSearchFieldsCarryTheID pins the sparse fieldset the request itself
// carries. The selection is applied locally as well, but it is forwarded so the
// smaller body is won on the wire wherever the endpoint reads the parameter —
// and an endpoint that reads it and was handed the caller's list verbatim
// answers rows with no identifier, which nothing downstream can put back. The
// mock answers the same body either way.
func TestTicketSearchFieldsCarryTheID(t *testing.T) {
	for _, tt := range []struct {
		name   string
		fields any
		want   string
	}{
		{name: "id appended", fields: []string{"subject", "createdAt"}, want: "subject,createdAt,id"},
		{name: "id not duplicated", fields: []string{"id", "subject"}, want: "id,subject"},
		{name: "no selection sends nothing", fields: nil, want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t,
				http.StatusOK, []byte(`{"tickets":[]}`))
			defer cleanup()

			testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketSearch.String(),
				searchArgs(map[string]any{"fields": tt.fields}))

			requestURL := lastRequestURL()
			if got := requestURL.Query().Get("fields"); got != tt.want {
				t.Errorf("query parameter \"fields\": got %q, want %q", got, tt.want)
			}
		})
	}
}
