package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestRateUserGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateUserGet.String()
	request.Params.Arguments = map[string]any{
		"id":                          float64(123),
		"page":                        float64(1),
		"page_size":                   float64(10),
		"include_installation_rate":   true,
		"include_user_cost":           true,
		"include_archived_projects":   false,
		"include_deleted_projects":    false,
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateInstallationUserList(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateInstallationUserList.String()
	request.Params.Arguments = map[string]any{
		"page":      float64(1),
		"page_size": float64(25),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateInstallationUserGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateInstallationUserGet.String()
	request.Params.Arguments = map[string]any{
		"user_id": float64(456),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateProjectGet.String()
	request.Params.Arguments = map[string]any{
		"project_id": float64(789),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectUserList(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateProjectUserList.String()
	request.Params.Arguments = map[string]any{
		"project_id":  float64(789),
		"search_term": "john",
		"page":        float64(1),
		"page_size":   float64(20),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectUserGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateProjectUserGet.String()
	request.Params.Arguments = map[string]any{
		"project_id": float64(789),
		"user_id":    float64(456),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectUserHistoryGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateProjectUserHistoryGet.String()
	request.Params.Arguments = map[string]any{
		"project_id": float64(789),
		"user_id":    float64(456),
		"page":       float64(1),
		"page_size":  float64(15),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateInstallationUserUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"STATUS":"OK"}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodRateInstallationUserUpdate.String()
	request.Params.Arguments = map[string]any{
		"user_id":     float64(456),
		"user_rate":   float64(75),
		"currency_id": float64(1),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateInstallationUserBulkUpdate(t *testing.T) {
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
	request.Params.Name = twprojects.MethodRateInstallationUserBulkUpdate.String()
	request.Params.Arguments = map[string]any{
		"user_rate":   float64(80),
		"all":         false,
		"ids":         []float64{456, 457, 458},
		"exclude_ids": []float64{},
		"currency_id": float64(1),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(`{"STATUS":"OK"}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodRateProjectUpdate.String()
	request.Params.Arguments = map[string]any{
		"project_id":   float64(789),
		"project_rate": float64(100),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectAndUsersUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(`{"STATUS":"OK"}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodRateProjectAndUsersUpdate.String()
	request.Params.Arguments = map[string]any{
		"project_id":   float64(789),
		"project_rate": float64(100),
		"user_rates": []map[string]any{
			{
				"user_id":   float64(456),
				"user_rate": float64(85),
			},
			{
				"user_id":   float64(457),
				"user_rate": float64(90),
			},
		},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestRateProjectUserUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"STATUS":"OK"}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodRateProjectUserUpdate.String()
	request.Params.Arguments = map[string]any{
		"project_id":  float64(789),
		"user_id":     float64(456),
		"user_rate":   float64(95),
		"currency_id": float64(1),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}