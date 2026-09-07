package twchat_test

import (
	"fmt"
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
	mcpServer, _ := dmMcpServerMock(t, 7,
		[]byte(`{"conversation":{"id":456,"type":"pair"},"STATUS":"ok"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodDMGetOrCreate.String(), map[string]any{
		"user_id": float64(42),
	})
}

func TestSendDM(t *testing.T) {
	// The fallback body serves both the get-or-create pair-conversation call
	// and the subsequent send-message call.
	mcpServer, _ := dmMcpServerMock(t, 7,
		[]byte(`{"conversation":{"id":456,"type":"pair"},"message":{"id":789},"STATUS":"ok"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodSendDM.String(), map[string]any{
		"user_id": float64(42),
		"body":    "Hello directly from MCP!",
	})
}

// TestDMToolsRejectTheCurrentUser pins the self-target guard on both direct
// message tools. Asserting the recorded requests is what makes it meaningful:
// the pair route creates the conversation as a side effect, so a rejection that
// still reached it would leave a self-conversation behind, and the mock would
// answer the send with the same body either way.
func TestDMToolsRejectTheCurrentUser(t *testing.T) {
	const currentUserID = 42

	for _, tt := range []struct {
		name string
		tool string
		args map[string]any
	}{{
		name: "send_dm",
		tool: twchat.MethodSendDM.String(),
		args: map[string]any{"user_id": float64(currentUserID), "body": "note to self"},
	}, {
		name: "get_or_create_dm",
		tool: twchat.MethodDMGetOrCreate.String(),
		args: map[string]any{"user_id": float64(currentUserID)},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := dmMcpServerMock(t, currentUserID,
				[]byte(`{"conversation":{"id":456,"type":"pair"},"message":{"id":789}}`))

			testutil.ExecuteToolRequest(t, mcpServer, tt.tool, tt.args,
				testutil.ExecuteToolRequestWithCheckMessage(checkToolError(t, "authenticated user")))

			for _, request := range *recorded {
				if path := request.URL.Path; path != "/chat/v7/me" {
					t.Errorf("rejected call still reached %s %s", request.Method, path)
				}
			}
		})
	}
}

// TestSendDMWithoutACurrentUserIDIsAnError pins the shape dependency: a /me
// payload carrying no identifiable user id must fail the call rather than
// silently disable the guard.
func TestSendDMWithoutACurrentUserIDIsAnError(t *testing.T) {
	mcpServer, recorded := testutil.ChatMCPServerRecordingMock(t, []testutil.ChatMockRoute{{
		Match:  "/chat/v7/me",
		Status: http.StatusOK,
		Body:   []byte(`{"account":{"firstName":"Jane"}}`),
	}}, http.StatusOK, []byte(`{"conversation":{"id":456},"message":{"id":789}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodSendDM.String(), map[string]any{
		"user_id": float64(42),
		"body":    "Hello directly from MCP!",
	}, testutil.ExecuteToolRequestWithCheckMessage(checkToolError(t, "no user id")))

	if len(*recorded) != 1 {
		t.Errorf("expected the call to stop after /me, got %d requests", len(*recorded))
	}
}

// TestCurrentUserIDMatchesTheLivePayload pins the parse against the shape the
// endpoint actually returns: the identity is nested under account.user, and the
// account repeats the same id at its top level. Values here are sanitized.
func TestCurrentUserIDMatchesTheLivePayload(t *testing.T) {
	body := []byte(`{"account":{"region":"US","user":{"id":12345,"firstName":"John","lastName":"Doe",` +
		`"email":"john@example.com","handle":"johndoe","role":"admin","deleted":false,` +
		`"apiKey":"twp_example","authKey":"tw-example-12345","installationName":"Example"},` +
		`"settings":{"timeFormat":"12hr"},"firstName":"John","lastName":"Doe","id":12345,` +
		`"authkey":"tw-example-12345","baseHref":"https://test.teamwork.com/","isAdmin":true,` +
		`"apiKey":"twp_example","installationId":777,"counts":{}},"status":"ok"}`)

	// Self-target: the id under account.user must be recognised as the caller.
	mcpServer, recorded := testutil.ChatMCPServerRecordingMock(t, []testutil.ChatMockRoute{{
		Match:  "/chat/v7/me",
		Status: http.StatusOK,
		Body:   body,
	}}, http.StatusOK, []byte(`{"conversation":{"id":456},"message":{"id":789}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodSendDM.String(), map[string]any{
		"user_id": float64(12345),
		"body":    "note to self",
	}, testutil.ExecuteToolRequestWithCheckMessage(checkToolError(t, "authenticated user")))

	if len(*recorded) != 1 {
		t.Errorf("expected the call to stop after /me, got %d requests", len(*recorded))
	}

	// Somebody else: the same payload must let the send through.
	mcpServer, recorded = testutil.ChatMCPServerRecordingMock(t, []testutil.ChatMockRoute{{
		Match:  "/chat/v7/me",
		Status: http.StatusOK,
		Body:   body,
	}}, http.StatusOK, []byte(`{"conversation":{"id":456},"message":{"id":789}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twchat.MethodSendDM.String(), map[string]any{
		"user_id": float64(54321),
		"body":    "Hello directly from MCP!",
	})

	if len(*recorded) != 3 {
		t.Errorf("expected /me, the pair lookup and the send, got %d requests", len(*recorded))
	}
}

// dmMcpServerMock answers /chat/v7/me with the given user id and everything
// else with fallbackBody, which is what the two-call direct message tools need.
func dmMcpServerMock(t *testing.T, currentUserID int64, fallbackBody []byte) (*mcp.Server, *[]testutil.ChatRecordedRequest) {
	t.Helper()

	return testutil.ChatMCPServerRecordingMock(t, []testutil.ChatMockRoute{{
		Match:  "/chat/v7/me",
		Status: http.StatusOK,
		Body:   []byte(fmt.Sprintf(`{"account":{"id":%d}}`, currentUserID)),
	}}, http.StatusOK, fallbackBody)
}

// checkToolError asserts the tool answered with an IsError result whose text
// contains want.
func checkToolError(t *testing.T, want string) func(*testing.T, mcp.Result) {
	t.Helper()

	return func(t *testing.T, result mcp.Result) {
		t.Helper()

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Fatalf("expected an error result, got: %v", toolResult.Content)
		}
		text, ok := toolResult.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type: %T", toolResult.Content[0])
		}
		if !strings.Contains(text.Text, want) {
			t.Errorf("expected error to mention %q, got: %s", want, text.Text)
		}
	}
}
