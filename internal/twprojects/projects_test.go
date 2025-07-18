package twprojects_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectCreate(t *testing.T) {
	mcpServer := mcpServerMock(t)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectCreate.String()
	request.Params.Arguments = map[string]any{
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	ctx := context.Background()
	message := mcpServer.HandleMessage(ctx, encodedRequest)
	if err, ok := message.(mcp.JSONRPCError); ok {
		t.Errorf("tool failed to execute: %v", err.Error)
	}
}

func TestProjectUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	ctx := context.Background()
	message := mcpServer.HandleMessage(ctx, encodedRequest)
	if err, ok := message.(mcp.JSONRPCError); ok {
		t.Errorf("tool failed to execute: %v", err.Error)
	}
}

func TestProjectDelete(t *testing.T) {
	mcpServer := mcpServerMock(t)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectDelete.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	ctx := context.Background()
	message := mcpServer.HandleMessage(ctx, encodedRequest)
	if err, ok := message.(mcp.JSONRPCError); ok {
		t.Errorf("tool failed to execute: %v", err.Error)
	}
}

func TestProjectGet(t *testing.T) {
	mcpServer := mcpServerMock(t)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectGet.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	ctx := context.Background()
	message := mcpServer.HandleMessage(ctx, encodedRequest)
	if err, ok := message.(mcp.JSONRPCError); ok {
		t.Errorf("tool failed to execute: %v", err.Error)
	}
}

func TestProjectList(t *testing.T) {
	mcpServer := mcpServerMock(t)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodProjectList.String()
	request.Params.Arguments = map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	ctx := context.Background()
	message := mcpServer.HandleMessage(ctx, encodedRequest)
	if err, ok := message.(mcp.JSONRPCError); ok {
		t.Errorf("tool failed to execute: %v", err.Error)
	}
}
