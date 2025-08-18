//nolint:lll
package twdesk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestMessageCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"message":{"id":123,"subject":"Test Message","body":"This is a test message"}}`))
	defer cleanup()

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twdesk.MethodMessageCreate.String()
	request.Params.Arguments = map[string]any{
		"ticketID": float64(456),
		"body":     "This is a test message",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	// This method is not implemented yet, so we expect it to fail
	// When it's implemented, change this to use checkMessage instead
	message := mcpServer.HandleMessage(context.Background(), encodedRequest)
	switch m := message.(type) {
	case mcp.JSONRPCResponse:
		if toolResult, ok := m.Result.(mcp.CallToolResult); ok {
			if !toolResult.IsError {
				t.Errorf("expected tool to fail (not implemented), but it succeeded")
			}
		}
	case mcp.JSONRPCError:
		// Expected - tool is not implemented
		if m.Error.Message != "not implemented" {
			t.Errorf("expected 'not implemented' error, got: %v", m.Error.Message)
		}
	default:
		t.Errorf("unexpected message type: %T", m)
	}
}
