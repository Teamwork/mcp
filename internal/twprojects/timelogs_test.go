package twprojects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTimelogCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timelog":{"id":123}}`))

	request := &toolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = twprojects.MethodTimelogCreate.String()
	request.Params.Arguments = map[string]any{
		"description": "Example timelog description",
		"date":        "2023-12-31",
		"time":        "12:00:00",
		"is_utc":      true,
		"hours":       float64(1),
		"minutes":     float64(30),
		"billable":    true,
		"project_id":  float64(123),
		"task_id":     float64(456),
		"user_id":     float64(789),
		"tag_ids":     []float64{10, 11, 12},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogUpdate(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogUpdate.String()
	request.Params.Arguments = map[string]any{
		"id":          float64(123),
		"description": "Example timelog description",
		"date":        "2023-12-31",
		"time":        "12:00:00",
		"is_utc":      true,
		"hours":       float64(1),
		"minutes":     float64(30),
		"billable":    true,
		"project_id":  float64(123),
		"task_id":     float64(456),
		"user_id":     float64(789),
		"tag_ids":     []float64{10, 11, 12},
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogDelete(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogDelete.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogGet(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogGet.String()
	request.Params.Arguments = map[string]any{
		"id": float64(123),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogList(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogList.String()
	request.Params.Arguments = map[string]any{
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogListByProject(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogListByProject.String()
	request.Params.Arguments = map[string]any{
		"project_id":           float64(123),
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}

func TestTimelogListByTask(t *testing.T) {
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
	request.Params.Name = twprojects.MethodTimelogListByTask.String()
	request.Params.Arguments = map[string]any{
		"task_id":              float64(123),
		"tag_ids":              []float64{1, 2, 3},
		"match_all_tags":       true,
		"start_date":           "2023-01-01T00:00:00Z",
		"end_date":             "2023-12-31T23:59:59Z",
		"assigned_user_ids":    []float64{1, 2, 3},
		"assigned_company_ids": []float64{4, 5, 6},
		"assigned_team_ids":    []float64{7, 8, 9},
		"page":                 float64(1),
		"page_size":            float64(10),
	}

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	checkMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
