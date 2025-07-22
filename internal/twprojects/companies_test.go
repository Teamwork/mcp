package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCompanyCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"company":{"id":123}}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodCompanyCreate.String()
	request.Params.Arguments = map[string]any{
		"name":         "Example",
		"address_one":  "123 Example St",
		"address_two":  "Suite 456",
		"city":         "Example City",
		"state":        "EX",
		"zip":          "12345",
		"country_code": "US",
		"phone":        "123-456-7890",
		"fax":          "098-765-4321",
		"email_one":    "example1@test.com",
		"email_two":    "example2@test.com",
		"email_three":  "example3@test.com",
		"website":      "https://www.example.com",
		"profile":      "Example Company Profile",
		"manager_id":   float64(456),
		"industry_id":  float64(789),
		"tag_ids":      []float64{1, 2, 3},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyUpdate(t *testing.T) {
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
	request.Params.Name = twprojects.MethodCompanyUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":           float64(123),
		"name":         "Example",
		"address_one":  "123 Example St",
		"address_two":  "Suite 456",
		"city":         "Example City",
		"state":        "EX",
		"zip":          "12345",
		"country_code": "US",
		"phone":        "123-456-7890",
		"fax":          "098-765-4321",
		"email_one":    "example1@test.com",
		"email_two":    "example2@test.com",
		"email_three":  "example3@test.com",
		"website":      "https://www.example.com",
		"profile":      "Example Company Profile",
		"manager_id":   float64(456),
		"industry_id":  float64(789),
		"tag_ids":      []float64{1, 2, 3},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodCompanyDelete.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodCompanyGet.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestCompanyList(t *testing.T) {
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
	request.Params.Name = twprojects.MethodCompanyList.String()
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
