package twchat_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twchat"
)

func TestCurrentUserGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"account":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodCurrentUserGet.String(), map[string]any{})
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
