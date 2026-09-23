package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCreate.String(), map[string]any{
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
	})
}

func TestProjectUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Example",
		"description": "This is an example project.",
		"start_at":    "20230101",
		"end_at":      "20231231",
		"company_id":  float64(123),
		"owner_id":    float64(456),
		"tag_ids":     []float64{1, 2, 3},
		"status":      "archived",
	})
}

func TestProjectTimelogRequiresTaskReachesTheWire(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		status    int
		arguments map[string]any
		want      any
		absent    bool
	}{{
		name:      "create not named",
		method:    twprojects.MethodProjectCreate.String(),
		status:    http.StatusCreated,
		arguments: map[string]any{"name": "Example"},
		want:      false,
	}, {
		name:      "create set",
		method:    twprojects.MethodProjectCreate.String(),
		status:    http.StatusCreated,
		arguments: map[string]any{"name": "Example", "timelog_requires_task": true},
		want:      true,
	}, {
		name:      "update not named",
		method:    twprojects.MethodProjectUpdate.String(),
		status:    http.StatusOK,
		arguments: map[string]any{"id": float64(123), "name": "Example"},
		absent:    true,
	}, {
		name:      "update set",
		method:    twprojects.MethodProjectUpdate.String(),
		status:    http.StatusOK,
		arguments: map[string]any{"id": float64(123), "timelog_requires_task": true},
		want:      true,
	}, {
		name:      "update cleared",
		method:    twprojects.MethodProjectUpdate.String(),
		status:    http.StatusOK,
		arguments: map[string]any{"id": float64(123), "timelog_requires_task": false},
		want:      false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, body := mcpServerMockWithRequestBody(t, tt.status, []byte(`{"id":"123"}`))
			testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.arguments)

			var payload struct {
				Project map[string]any `json:"project"`
			}
			if err := json.Unmarshal(*body, &payload); err != nil {
				t.Fatalf("failed to decode request body: %s", err)
			}
			got, ok := payload.Project["timelogRequiresTask"]
			switch {
			case tt.absent && ok:
				t.Errorf("expected timelogRequiresTask to be absent from the request body, got %v", got)
			case !tt.absent && !ok:
				t.Errorf("expected timelogRequiresTask in the request body, got %v", payload.Project)
			case !tt.absent && got != tt.want:
				t.Errorf("expected timelogRequiresTask to be %v, got %v", tt.want, got)
			}
		})
	}
}

func TestProjectDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectClone(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"projectId":123}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectClone.String(), map[string]any{
		"id":                   float64(123),
		"name":                 "Cloned Project",
		"description":          "This is a cloned project.",
		"company_id":           float64(123),
		"new_from_template":    true,
		"to_template":          false,
		"template_date_target": "start",
		"target_date":          "20240101",
		"days_offset":          float64(7),
	})
}

func TestProjectGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestProjectList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
