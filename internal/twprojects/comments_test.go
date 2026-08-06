package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCommentCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreate.String(), map[string]any{
		"object": map[string]any{
			"type": "tasks",
			"id":   float64(123),
		},
		"body":                "Example",
		"content_type":        "TEXT",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []any{float64(1), float64(2)},
			"team_ids":    []any{float64(3)},
			"company_ids": []any{float64(4)},
		},
	})
}

// Comment-specific notify contract: true (and the default) mean followers;
// the shared shapes behave as in the other notify-carrying tools.
func TestCommentCreateNotifyShapes(t *testing.T) {
	tests := []struct {
		name       string
		notify     any
		wantNotify string // raw JSON as serialized for the API; empty means omitted
	}{
		{name: "boolean true notifies followers", notify: true, wantNotify: `true`},
		{name: "boolean false notifies nobody", notify: false, wantNotify: ""},
		{name: "explicit null falls back to followers", notify: nil, wantNotify: `true`},
		{name: "string all notifies project members", notify: "all", wantNotify: `"ALL"`},
		{name: "array of user IDs", notify: []any{float64(1), float64(2)}, wantNotify: `"1,2"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated, []byte(`{"id":"123"}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreate.String(), map[string]any{
				"object": map[string]any{
					"type": "tasks",
					"id":   float64(123),
				},
				"body":   "Example",
				"notify": tt.notify,
			})

			var payload struct {
				Comment struct {
					Notify json.RawMessage `json:"notify"`
				} `json:"comment"`
			}
			if err := json.Unmarshal(*requestBody, &payload); err != nil {
				t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
			}
			if string(payload.Comment.Notify) != tt.wantNotify {
				t.Errorf("expected notify %s, got %s (body %q)",
					tt.wantNotify, payload.Comment.Notify, string(*requestBody))
			}
		})
	}
}

func TestCommentUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentUpdate.String(), map[string]any{
		"id":                  float64(123),
		"body":                "Example",
		"content_type":        "TEXT",
		"notify_current_user": true,
		"notify":              "all",
	})
}

func TestCommentDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByFileVersion(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":     "test",
		"file_version_id": float64(123),
		"updated_after":   "2025-01-01T00:00:00Z",
		"page":            float64(1),
		"page_size":       float64(10),
	})
}

func TestCommentListByMilestone(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"milestone_id":  float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByNotebook(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"notebook_id":   float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByTask(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"task_id":       float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByLink(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"link_id":       float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}
