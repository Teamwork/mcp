//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestInboxGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inbox":{"id":123,"name":"Support Inbox","email":"support@example.com","description":"Main support inbox"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxGet.String(), map[string]any{
		"id": "123",
	})
}

func TestInboxList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":123,"name":"Support Inbox","email":"support@example.com"},{"id":124,"name":"Sales Inbox","email":"sales@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":     []string{"Support Inbox", "Sales Inbox"},
		"email":    []string{"support@example.com"},
		"page":     float64(1),
		"pageSize": float64(10),
	})
}

func TestInboxListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{})
}

func TestInboxListWithNameFilter(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":123,"name":"Support Inbox","email":"support@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name": []string{"Support Inbox"},
	})
}

func TestInboxListWithEmailFilter(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":124,"name":"Sales Inbox","email":"sales@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"email": []string{"sales@example.com"},
	})
}

func TestInboxListWithPagination(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":125,"name":"General Inbox","email":"general@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"page":     float64(2),
		"pageSize": float64(5),
	})
}
