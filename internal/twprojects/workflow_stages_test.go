package twprojects_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestWorkflowStageCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"stage":{"id":456,"name":"In Progress"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageCreate.String(), map[string]any{
		"workflow_id": float64(123),
		"name":        "In Progress",
	})
}

func TestWorkflowStageUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageUpdate.String(), map[string]any{
		"workflow_id": float64(123),
		"id":          float64(456),
		"name":        "Updated Stage",
	})
}

func TestWorkflowStageDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageDelete.String(), map[string]any{
		"workflow_id":           float64(123),
		"id":                    float64(456),
		"map_tasks_to_stage_id": float64(789),
	})
}

// Pins the transport, not the outcome: the bulk route answers 204 too, but 403s
// for non-administrators.
func TestWorkflowStageTaskMoveUsesTheBoardRoute(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageTaskMove.String(), map[string]any{
		"workflow_id": float64(123),
		"stage_id":    float64(456),
		"task_ids":    []float64{789, 790, 791},
	})

	if len(*recorded) != 3 {
		t.Fatalf("expected one HTTP request per task, got %d", len(*recorded))
	}
	for i, want := range []int64{789, 790, 791} {
		entry := (*recorded)[i]
		if entry.Method != http.MethodPatch {
			t.Errorf("request %d: expected PATCH, got %s", i, entry.Method)
		}
		wantPath := fmt.Sprintf("/projects/api/v3/tasks/%d/workflows/123.json", want)
		if entry.URL.Path != wantPath {
			t.Errorf("request %d: expected path %s, got %s", i, wantPath, entry.URL.Path)
		}

		var payload struct {
			StageID           int64 `json:"stageId"`
			PositionAfterTask int64 `json:"positionAfterTask"`
		}
		if err := json.Unmarshal(entry.Body, &payload); err != nil {
			t.Fatalf("request %d: failed to decode body %q: %v", i, string(entry.Body), err)
		}
		if payload.StageID != 456 {
			t.Errorf("request %d: expected stageId 456, got %d", i, payload.StageID)
		}
		// -1 appends, so the tasks keep the order given.
		if payload.PositionAfterTask != -1 {
			t.Errorf("request %d: expected positionAfterTask -1, got %d", i, payload.PositionAfterTask)
		}
	}
}

// Clients holding the tool list from before this tool took a set.
func TestWorkflowStageTaskMoveAcceptsLegacyTaskID(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageTaskMove.String(), map[string]any{
		"workflow_id": float64(123),
		"stage_id":    float64(456),
		"task_id":     float64(789),
	})

	if len(*recorded) != 1 {
		t.Fatalf("expected a single HTTP request, got %d", len(*recorded))
	}
	if want := "/projects/api/v3/tasks/789/workflows/123.json"; (*recorded)[0].URL.Path != want {
		t.Errorf("expected path %s, got %s", want, (*recorded)[0].URL.Path)
	}
}

func TestWorkflowStageTaskMoveRequiresTaskIDs(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageTaskMove.String(), map[string]any{
		"workflow_id": float64(123),
		"stage_id":    float64(456),
		"task_ids":    []float64{},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error when no task ID is provided")
		}
	}))
}

func TestWorkflowStageGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"stage":{"id":456,"name":"In Progress"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageGet.String(), map[string]any{
		"workflow_id": float64(123),
		"id":          float64(456),
	})
}

func TestWorkflowStageList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"stages":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageList.String(), map[string]any{
		"workflow_id": float64(123),
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
