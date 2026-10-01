package twdesk_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

// TestDeskResponsesDropInternalKeysOnly pins what every caller loses: the
// agent's ldKey and the included sections that hold nothing. Contact details
// stay; removing those is the restricted mode's job, not every caller's.
func TestDeskResponsesDropInternalKeysOnly(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		args    map[string]any
		body    string
		want    []string
		notWant []string
	}{{
		name:    "get_user",
		method:  twdesk.MethodUserGet.String(),
		args:    map[string]any{"id": float64(12345)},
		body:    `{"user":{"id":12345,"email":"jane@example.com","ldKey":"opaque-key"}}`,
		want:    []string{`"email":"jane@example.com"`},
		notWant: []string{"ldKey", "opaque-key", `"included"`},
	}, {
		name:   "get_ticket sideloads",
		method: twdesk.MethodTicketGet.String(),
		args:   map[string]any{"id": float64(1)},
		body: `{"ticket":{"id":1},"included":{"customers":[{"id":777,"phone":"+1 555 0100"}],` +
			`"users":[{"id":12345,"ldKey":"opaque-key"}]}}`,
		want:    []string{`"phone":"+1 555 0100"`, `"customers":[`},
		notWant: []string{"ldKey", `"tags":null`},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(tt.body))
			defer cleanup()
			testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.args,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok || toolResult.IsError {
						t.Fatalf("unexpected result: %#v", result)
					}
					text := toolResult.Content[0].(*mcp.TextContent).Text
					for _, s := range tt.want {
						if !strings.Contains(text, s) {
							t.Errorf("expected %s in %s", s, text)
						}
					}
					for _, s := range tt.notWant {
						if strings.Contains(text, s) {
							t.Errorf("%s reached the caller: %s", s, text)
						}
					}
				}))
		})
	}
}
