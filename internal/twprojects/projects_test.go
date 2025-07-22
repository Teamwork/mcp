package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))

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

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestProjectUpdate(t *testing.T) {
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

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestProjectDelete(t *testing.T) {
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
	request.Params.Name = twprojects.MethodProjectDelete.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestProjectGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodProjectGet.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestProjectList(t *testing.T) {
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

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
