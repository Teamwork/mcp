package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestActivityList(t *testing.T) {
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
	request.Params.Name = twprojects.MethodActivityList.String()
	request.Params.Arguments = map[string]any{
		"start_date": "2023-10-01T00:00:00Z",
		"end_date":   "2023-10-31T23:59:59Z",
		"log_item_types": []any{
			"message",
			"comment",
			"task",
			"tasklist",
			"taskgroup",
			"milestone",
			"file",
			"form",
			"notebook",
			"timelog",
			"task_comment",
			"notebook_comment",
			"file_comment",
			"link_comment",
			"milestone_comment",
			"project",
			"link",
			"billingInvoice",
			"risk",
			"projectUpdate",
			"reacted",
			"budget",
		},
		"page":      float64(1),
		"page_size": float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestActivityListByProject(t *testing.T) {
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
	request.Params.Name = twprojects.MethodActivityListByProject.String()
	request.Params.Arguments = map[string]any{
		"project_id": 123,
		"start_date": "2023-10-01T00:00:00Z",
		"end_date":   "2023-10-31T23:59:59Z",
		"log_item_types": []any{
			"message",
			"comment",
			"task",
			"tasklist",
			"taskgroup",
			"milestone",
			"file",
			"form",
			"notebook",
			"timelog",
			"task_comment",
			"notebook_comment",
			"file_comment",
			"link_comment",
			"milestone_comment",
			"project",
			"link",
			"billingInvoice",
			"risk",
			"projectUpdate",
			"reacted",
			"budget",
		},
		"page":      float64(1),
		"page_size": float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
