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
		"subject":    "Test Ticket",
		"body":       "This is a test ticket",
		"priorityId": "1",
		"statusId":   "1",
		"typeId":     "1",
		"customerId": "100",
		"inboxId":    "1",
		"agentId":    "1",
	})
}

func TestTicketUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Updated Ticket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketUpdate.String(), map[string]any{
		"id":         "123",
		"subject":    "Updated Ticket",
		"priorityId": "2",
		"statusId":   "2",
		"typeId":     "2",
	})
}

func TestTicketGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Test Ticket"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketGet.String(), map[string]any{
		"id": "123",
	})
}

func TestTicketList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"},{"id":124,"subject":"Ticket 2"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketList.String(), map[string]any{
		"statusIDs":   []float64{1, 2},
		"priorityIDs": []float64{1, 2, 3},
		"page":        float64(1),
		"pageSize":    float64(10),
	})
}

func TestTicketListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketList.String(), map[string]any{})
}

func TestTicketSearch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"},{"id":124,"subject":"Ticket 2"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTicketList.String(), map[string]any{
		"search":      "Testing 123",
		"statusIDs":   []float64{1, 2},
		"priorityIDs": []float64{1, 2, 3},
		"page":        float64(1),
		"pageSize":    float64(10),
	})
}
