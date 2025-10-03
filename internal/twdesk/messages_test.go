//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestMessageCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"message":{"id":123,"subject":"Test Message","body":"This is a test message"}}`))
	defer cleanup()

	// This method is not implemented yet, so we expect it to fail
	// When it's implemented, change this to use checkMessage instead
	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodMessageCreate.String(), map[string]any{
		"ticketID": float64(456),
		"body":     "This is a test message",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Errorf("unexpected result type: %T", result)
			return
		}
		if !toolResult.IsError {
			t.Errorf("expected tool to fail (not implemented), but it succeeded")
		}
	}))
}
