package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestMessageReplyCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"postId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyCreate.String(), map[string]any{
		"message_id":          float64(123),
		"body":                "Example message reply body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

// Covers every accepted notify shape, including coerced near-misses, by
// asserting the notifier actually serialized for the API.
func TestMessageReplyCreateNotifyShapes(t *testing.T) {
	tests := []struct {
		name       string
		notify     any
		wantNotify string // raw JSON as serialized for the API; empty means omitted
	}{
		{name: "string all", notify: "all", wantNotify: `"ALL"`},
		{name: "array of user IDs", notify: []any{float64(1), float64(2)}, wantNotify: `"1,2"`},
		{name: "boolean true coerces to all", notify: true, wantNotify: `"ALL"`},
		{name: "boolean false notifies nobody", notify: false, wantNotify: ""},
		{name: "explicit null falls back to the default", notify: nil, wantNotify: `"ALL"`},
		{
			name: "object of user groups",
			notify: map[string]any{
				"user_ids": []any{float64(1), float64(2)},
				"team_ids": []any{float64(6)},
			},
			wantNotify: `"1,2,t6"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated, []byte(`{"postId":"123"}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyCreate.String(), map[string]any{
				"message_id": float64(123),
				"body":       "Example message reply body",
				"notify":     tt.notify,
			})

			var payload struct {
				MessageReply struct {
					Notify json.RawMessage `json:"notify"`
				} `json:"messagereply"`
			}
			if err := json.Unmarshal(*requestBody, &payload); err != nil {
				t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
			}
			if string(payload.MessageReply.Notify) != tt.wantNotify {
				t.Errorf("expected notify %s, got %s (body %q)",
					tt.wantNotify, payload.MessageReply.Notify, string(*requestBody))
			}
		})
	}
}

// Shapes outside the contract are rejected by SDK schema validation before
// reaching the API; the handler's fallback error is pinned in helpers_notify_test.go.
func TestMessageReplyCreateNotifyRejectsUnknownShapes(t *testing.T) {
	for _, notify := range []any{"everyone", float64(7), []any{"bob"}, []any{}} {
		mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"postId":"123"}`))
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyCreate.String(), map[string]any{
			"message_id": float64(123),
			"body":       "Example message reply body",
			"notify":     notify,
		}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if !toolResult.IsError {
				t.Fatalf("expected notify %#v to be rejected", notify)
			}
		}))
	}
}

func TestMessageReplyUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyUpdate.String(), map[string]any{
		"id":                  float64(123),
		"body":                "Example message reply body",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestMessageReplyDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageReplyGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestMessageReplyList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageReplyList.String(), map[string]any{
		"search_term": "test",
		"message_ids": []float64{1, 2, 3},
		"project_ids": []float64{1, 2, 3},
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
