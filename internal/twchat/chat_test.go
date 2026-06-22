package twchat_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twchat"
)

func TestCurrentUserGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"account":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodCurrentUserGet.String(), map[string]any{})
}

func TestCurrentUserGetRedactsCredentials(t *testing.T) {
	body := []byte(`{"account":{"apiKey":"twp_secret","authkey":"tkn_secret",` +
		`"user":{"id":1,"apiKey":"twp_secret","authKey":"tkn_secret"}},"status":"ok"}`)
	mcpServer := mcpServerMock(t, http.StatusOK, body)
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodCurrentUserGet.String(), map[string]any{},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if toolResult.IsError {
				t.Fatalf("tool returned an error: %v", toolResult.Content)
			}
			if len(toolResult.Content) != 1 {
				t.Fatalf("expected 1 content item, got %d", len(toolResult.Content))
			}
			text, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("unexpected content type: %T", toolResult.Content[0])
			}
			for _, secret := range []string{"apiKey", "authKey", "authkey", "twp_secret", "tkn_secret"} {
				if strings.Contains(text.Text, secret) {
					t.Errorf("expected %q to be redacted, but it is present in: %s", secret, text.Text)
				}
			}
			// Non-sensitive fields must survive redaction.
			if !strings.Contains(text.Text, `"status":"ok"`) {
				t.Errorf("expected non-sensitive fields to be preserved, got: %s", text.Text)
			}
		}))
}

func TestConversationList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"conversations":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodConversationList.String(), map[string]any{
		"search_term":          "design",
		"status":               "active",
		"sort":                 "lastActivityAt",
		"include_message_data": true,
		"page_offset":          float64(0),
		"page_limit":           float64(5),
	})
}

func TestConversationGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"conversation":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodConversationGet.String(), map[string]any{
		"conversation_id": float64(123),
	})
}

func TestMessageList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"messages":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodMessageList.String(), map[string]any{
		"conversation_id":   float64(123),
		"search_term":       "release",
		"page":              float64(1),
		"page_size":         float64(50),
		"before_message_id": float64(999),
		"after_message_id":  float64(100),
		"created_before":    "2023-12-31T23:59:59Z",
		"created_after":     "2023-01-01T00:00:00Z",
	})
}

func TestPeopleList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"people":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodPeopleList.String(), map[string]any{
		"search_term": "jane",
		"page_offset": float64(0),
		"page_limit":  float64(10),
	})
}

func TestMessageSend(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"id":"789","message":{"id":789}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodMessageSend.String(), map[string]any{
		"conversation_id": float64(123),
		"body":            "Hello from MCP!",
	})
}

func TestConversationListByType(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"conversations":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodConversationList.String(), map[string]any{
		"type": "pair",
	})
}

func TestDMGetOrCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"conversation":{"id":456,"type":"pair"},"STATUS":"ok"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodDMGetOrCreate.String(), map[string]any{
		"user_id": float64(42),
	})
}

func TestSendDM(t *testing.T) {
	// The single-response mock returns this body for both the get-or-create
	// pair-conversation call and the subsequent send-message call.
	mcpServer := mcpServerMock(t, http.StatusOK,
		[]byte(`{"conversation":{"id":456,"type":"pair"},"message":{"id":789},"STATUS":"ok"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodSendDM.String(), map[string]any{
		"user_id": float64(42),
		"body":    "Hello directly from MCP!",
	})
}
