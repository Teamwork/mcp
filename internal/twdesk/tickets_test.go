//nolint:lll
package twdesk_test

import (
	"net/http"
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
