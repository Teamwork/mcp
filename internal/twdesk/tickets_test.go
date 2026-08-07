//nolint:lll
package twdesk_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
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
		"page":           float64(3),
		"pageSize":       float64(250),
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
		"pageSize":  "250",
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
	for _, key := range []string{"orderBy", "orderMode", "fields"} {
		if query.Has(key) {
			t.Errorf("query parameter %q should not be sent when not requested, got %q", key, query.Get(key))
		}
	}
}
