//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestMessageCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"message":{"id":123,"subject":"Test Message","body":"This is a test message"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodMessageCreate.String(), map[string]any{
		"ticketID": float64(456),
		"body":     "This is a test message",
		"cc":       []string{"cc@example.com"},
		"bcc":      []string{"bcc@example.com"},
	})
}
