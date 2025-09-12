package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestUsersWorkload(t *testing.T) {
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
	request.Params.Name = twprojects.MethodUsersWorkload.String()
	request.Params.Arguments = map[string]any{
		"start_date":       "2023-01-01",
		"end_date":         "2023-01-31",
		"user_ids":         []float64{1, 2, 3},
		"user_company_ids": []float64{4, 5, 6},
		"user_team_ids":    []float64{7, 8, 9},
		"project_ids":      []float64{10, 11, 12},
		"page":             float64(1),
		"page_size":        float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
