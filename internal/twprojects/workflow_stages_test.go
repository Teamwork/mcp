package twprojects_test

import (
	"encoding/json"
	"net/http"
	"slices"
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

func TestWorkflowStageTaskMove(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusNoContent, []byte(``))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodWorkflowStageTaskMove.String(), map[string]any{
		"workflow_id": float64(123),
		"stage_id":    float64(456),
		"task_ids":    []float64{789, 790, 791},
	})

	// The whole set has to travel in one request; looping the single-task
	// endpoint instead is the call amplification this tool exists to avoid.
	if len(*recorded) != 1 {
		t.Fatalf("expected a single HTTP request, got %d", len(*recorded))
	}
	entry := (*recorded)[0]
	if entry.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", entry.Method)
	}
	if want := "/projects/api/v3/workflows/123/stages/456/tasks.json"; entry.Path != want {
		t.Errorf("expected path %s, got %s", want, entry.Path)
	}

	var payload struct {
		TaskIDs []int64 `json:"taskIds"`
	}
	if err := json.Unmarshal(entry.Body, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(entry.Body), err)
	}
	if want := []int64{789, 790, 791}; !slices.Equal(payload.TaskIDs, want) {
		t.Errorf("expected taskIds %v, got %v (body %q)", want, payload.TaskIDs, string(entry.Body))
	}
}

// TestWorkflowStageTaskMoveAcceptsLegacyTaskID covers clients still holding the
// tool list from before this tool took a set. The scalar is no longer
// advertised, but dropping it would break them mid-session.
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
	var payload struct {
		TaskIDs []int64 `json:"taskIds"`
	}
	if err := json.Unmarshal((*recorded)[0].Body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if want := []int64{789}; !slices.Equal(payload.TaskIDs, want) {
		t.Errorf("expected taskIds %v, got %v", want, payload.TaskIDs)
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
