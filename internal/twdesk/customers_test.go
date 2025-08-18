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

func TestCustomerCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"customer":{"id":123,"firstName":"John","lastName":"Doe","email":"john@example.com"}}`))
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
	request.Params.Name = twdesk.MethodCustomerCreate.String()
	request.Params.Arguments = map[string]any{
		"id":            "123",
		"firstName":     "John",
		"lastName":      "Doe",
		"email":         "john@example.com",
		"organization":  "Test Corp",
		"extraData":     "Some extra data",
		"notes":         "Test customer notes",
		"linkedinURL":   "https://linkedin.com/in/johndoe",
		"facebookURL":   "https://facebook.com/johndoe",
		"twitterHandle": "@johndoe",
		"jobTitle":      "Software Engineer",
		"phone":         "+1234567890",
		"mobile":        "+0987654321",
		"address":       "123 Test St, Test City",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCustomerUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customer":{"id":123,"firstName":"Jane","lastName":"Smith","email":"jane@example.com"}}`))
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
	request.Params.Name = twdesk.MethodCustomerUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":            "123",
		"firstName":     "Jane",
		"lastName":      "Smith",
		"email":         "jane@example.com",
		"organization":  "Updated Corp",
		"extraData":     "Updated extra data",
		"notes":         "Updated customer notes",
		"linkedinURL":   "https://linkedin.com/in/janesmith",
		"facebookURL":   "https://facebook.com/janesmith",
		"twitterHandle": "@janesmith",
		"jobTitle":      "Senior Engineer",
		"phone":         "+1111111111",
		"mobile":        "+2222222222",
		"address":       "456 Updated St, Updated City",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCustomerGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customer":{"id":123,"firstName":"John","lastName":"Doe","email":"john@example.com"}}`))
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
	request.Params.Name = twdesk.MethodCustomerGet.String()
	request.Params.Arguments = map[string]any{
		"id": "123",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCustomerList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customers":[{"id":123,"firstName":"John","lastName":"Doe"},{"id":124,"firstName":"Jane","lastName":"Smith"}]}`))
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
	request.Params.Name = twdesk.MethodCustomerList.String()
	request.Params.Arguments = map[string]any{
		"companyIDs":   []float64{1, 2, 3},
		"companyNames": []string{"Test Corp", "Example Inc"},
		"emails":       []string{"john@example.com", "jane@example.com"},
		"page":         float64(1),
		"page_size":    float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCustomerListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customers":[]}`))
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
	request.Params.Name = twdesk.MethodCustomerList.String()
	request.Params.Arguments = map[string]any{}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
