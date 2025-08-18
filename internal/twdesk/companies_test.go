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

func TestCompanyCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"company":{"id":123,"name":"Test Company","kind":"company"}}`))
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
	request.Params.Name = twdesk.MethodCompanyCreate.String()
	request.Params.Arguments = map[string]any{
		"id":          "123",
		"name":        "Test Company",
		"description": "A test company",
		"details":     "Company details",
		"industry":    "Technology",
		"website":     "https://example.com",
		"permission":  "own",
		"kind":        "company",
		"note":        "Test note",
		"domains":     []string{"example.com", "test.com"},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"company":{"id":123,"name":"Updated Company","kind":"company"}}`))
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
	request.Params.Name = twdesk.MethodCompanyUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":          "123",
		"name":        "Updated Company",
		"description": "Updated description",
		"details":     "Updated details",
		"industry":    "Software",
		"website":     "https://updated.com",
		"permission":  "all",
		"kind":        "group",
		"note":        "Updated note",
		"domains":     []string{"updated.com"},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"company":{"id":123,"name":"Test Company","kind":"company"}}`))
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
	request.Params.Name = twdesk.MethodCompanyGet.String()
	request.Params.Arguments = map[string]any{
		"id": "123",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"companies":[{"id":123,"name":"Company 1","kind":"company"},{"id":124,"name":"Company 2","kind":"group"}]}`))
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
	request.Params.Name = twdesk.MethodCompanyList.String()
	request.Params.Arguments = map[string]any{
		"name":      "Test Company",
		"domains":   []string{"example.com", "test.com"},
		"kind":      "company",
		"page":      float64(1),
		"page_size": float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"companies":[]}`))
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
	request.Params.Name = twdesk.MethodCompanyList.String()
	request.Params.Arguments = map[string]any{}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
