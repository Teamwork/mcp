package twdesk_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// TestAPIFailuresAreToolResults pins that a Desk API failure reaches the caller
// as an IsError tool result rather than as a Go error out of the handler.
//
// A raw error becomes a protocol-level error: the model sees a transport
// failure with no room to read the status and retry, and the handler's Go
// wrapping ("failed to get ticket: ...") never reaches it. Every Desk handler
// therefore routes API errors through helpers.HandleAPIError, the same as
// twprojects.
//
// The mechanism this guards is easy to lose because desksdkgo declares no error
// type: its non-2xx errors are bare fmt.Errorf values, so HandleAPIError has to
// read the status out of the message text. ExecuteToolRequest fails on a Go
// error before the assertion runs, which is what catches a regression here.
func TestAPIFailuresAreToolResults(t *testing.T) {
	tests := []struct {
		name   string
		method toolsets.Method
		args   map[string]any
	}{
		{
			name:   "get_ticket",
			method: twdesk.MethodTicketGet,
			args:   map[string]any{"id": float64(123), "fields": nil},
		},
		{
			name:   "search_tickets",
			method: twdesk.MethodTicketSearch,
			args: map[string]any{
				"search": "anything", "inboxIDs": nil, "customerIDs": nil, "companyIDs": nil,
				"tagIDs": nil, "statusIDs": nil, "priorityIDs": nil, "userIDs": nil,
				"createdAfter": nil, "createdBefore": nil,
				"page": nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil, "fields": nil,
			},
		},
		{
			name:   "link_task_to_ticket",
			method: twdesk.MethodTicketTaskLink,
			args:   map[string]any{"ticketId": float64(123), "taskId": float64(456)},
		},
		{
			name:   "unlink_task_from_ticket",
			method: twdesk.MethodTicketTaskUnlink,
			args:   map[string]any{"ticketId": float64(123), "taskId": float64(456)},
		},
		{
			name:   "get_inbox",
			method: twdesk.MethodInboxGet,
			args:   map[string]any{"id": float64(123), "fields": nil},
		},
		{
			name:   "list_inboxes",
			method: twdesk.MethodInboxList,
			args: map[string]any{
				"name": nil, "email": nil,
				"page": nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil, "fields": nil,
			},
		},
		{
			name:   "get_helpdoc_article",
			method: twdesk.MethodHelpDocArticleGet,
			args:   map[string]any{"id": float64(42), "fields": nil},
		},
		{
			name:   "search_helpdoc_articles",
			method: twdesk.MethodHelpDocArticleSearch,
			args: map[string]any{
				"search": nil, "status": nil, "siteID": nil, "categoryID": nil,
				"page": nil, "pageSize": nil,
			},
		},
		{
			name:   "create_tag",
			method: twdesk.MethodTagCreate,
			args:   map[string]any{"name": "urgent", "color": "red"},
		},
		{
			name:   "update_tag",
			method: twdesk.MethodTagUpdate,
			args:   map[string]any{"id": float64(123), "name": "important", "color": "orange"},
		},
	}

	statuses := []struct {
		status int
		want   string
	}{
		{status: http.StatusNotFound, want: "bad request"},
		{status: http.StatusForbidden, want: "bad request"},
		{status: http.StatusInternalServerError, want: "server error"},
	}

	for _, tt := range tests {
		for _, s := range statuses {
			t.Run(tt.name+"/"+http.StatusText(s.status), func(t *testing.T) {
				mcpServer, cleanup := mcpServerMock(t, s.status, []byte(`{"errors":[{"detail":"nope"}]}`))
				defer cleanup()

				testutil.ExecuteToolRequest(t, mcpServer, tt.method.String(), tt.args,
					testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
						t.Helper()

						toolResult, ok := result.(*mcp.CallToolResult)
						if !ok {
							t.Fatalf("unexpected result type: %T", result)
						}
						if !toolResult.IsError {
							t.Fatalf("HTTP %d should produce an error tool result", s.status)
						}
						if len(toolResult.Content) == 0 {
							t.Fatal("error tool result should carry content the model can read")
						}
						textContent, ok := toolResult.Content[0].(*mcp.TextContent)
						if !ok {
							t.Fatalf("unexpected content type: %T", toolResult.Content[0])
						}
						if !strings.Contains(textContent.Text, s.want) {
							t.Errorf("error text should classify HTTP %d as %q, got %q", s.status, s.want, textContent.Text)
						}
					}))
			})
		}
	}
}
