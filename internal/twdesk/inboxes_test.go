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
		"id":     float64(123),
		"fields": nil,
	})
}

func TestInboxList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":123,"name":"Support Inbox","email":"support@example.com"},{"id":124,"name":"Sales Inbox","email":"sales@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":           []string{"Support Inbox", "Sales Inbox"},
		"email":          []string{"support@example.com"},
		"page":           float64(1),
		"pageSize":       float64(10),
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestInboxListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":           nil,
		"email":          nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestInboxListWithNameFilter(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":123,"name":"Support Inbox","email":"support@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":           []string{"Support Inbox"},
		"email":          nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestInboxListWithEmailFilter(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":124,"name":"Sales Inbox","email":"sales@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":           nil,
		"email":          []string{"sales@example.com"},
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestInboxListWithPagination(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"inboxes":[{"id":125,"name":"General Inbox","email":"general@example.com"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodInboxList.String(), map[string]any{
		"name":           nil,
		"email":          nil,
		"page":           float64(2),
		"pageSize":       float64(5),
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}
