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

func TestTicketCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket":{"id":123,"subject":"Test Ticket"}}`))
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
	request.Params.Name = twdesk.MethodTicketCreate.String()
	request.Params.Arguments = map[string]any{
		"subject":    "Test Ticket",
		"body":       "This is a test ticket",
		"priorityId": "1",
		"statusId":   "1",
		"typeId":     "1",
		"customerId": "100",
		"inboxId":    "1",
		"agentId":    "1",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTicketUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Updated Ticket"}}`))
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
	request.Params.Name = twdesk.MethodTicketUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":         "123",
		"subject":    "Updated Ticket",
		"priorityId": "2",
		"statusId":   "2",
		"typeId":     "2",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTicketGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket":{"id":123,"subject":"Test Ticket"}}`))
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
	request.Params.Name = twdesk.MethodTicketGet.String()
	request.Params.Arguments = map[string]any{
		"id": "123",
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTicketList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"},{"id":124,"subject":"Ticket 2"}]}`))
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
	request.Params.Name = twdesk.MethodTicketList.String()
	request.Params.Arguments = map[string]any{
		"status_ids":   []float64{1, 2},
		"priority_ids": []float64{1, 2, 3},
		"page":         float64(1),
		"page_size":    float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTicketListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[]}`))
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
	request.Params.Name = twdesk.MethodTicketList.String()
	request.Params.Arguments = map[string]any{}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTicketSearch(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tickets":[{"id":123,"subject":"Ticket 1"},{"id":124,"subject":"Ticket 2"}]}`))
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
	request.Params.Name = twdesk.MethodTicketList.String()
	request.Params.Arguments = map[string]any{
		"search":       "Testing 123",
		"status_ids":   []float64{1, 2},
		"priority_ids": []float64{1, 2, 3},
		"page":         float64(1),
		"page_size":    float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
