package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectMemberAdd(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectMemberAdd.String()
	request.Params.Arguments = map[string]any{
		"project_id": float64(456),
		"user_ids":   []any{float64(123), float64(456)},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
