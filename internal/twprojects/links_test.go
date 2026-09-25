package twprojects_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestLinkCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkCreate.String(), map[string]any{
		"code":                "http://example.com",
		"project_id":          float64(123),
		"title":               "Example",
		"description":         "Example message body",
		"tag_ids":             []float64{1, 2, 3},
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestLinkUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkUpdate.String(), map[string]any{
		"id":                  float64(123),
		"code":                "http://example.com",
		"title":               "Example",
		"description":         "Example message body",
		"tag_ids":             []float64{1, 2, 3},
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []float64{1, 2, 3},
			"company_ids": []float64{4, 5},
			"team_ids":    []float64{6, 7},
		},
	})
}

func TestLinkDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestLinkGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestLinkList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodLinkList.String(), map[string]any{
		"search_term":    "test",
		"project_id":     float64(123),
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}

// An omitted notify must send no notification: the mocks answer the same body
// either way, so the value is asserted on the request body.
func TestLinkNotifyReachesTheWire(t *testing.T) {
	tests := []struct {
		name   string
		notify any
		want   string
	}{
		{name: "omitted", want: ""},
		{name: "false", notify: false, want: ""},
		{name: "all", notify: "all", want: `"ALL"`},
		{name: "true", notify: true, want: `"ALL"`},
		{name: "users", notify: []any{float64(777)}, want: `"777"`},
	}
	tools := []struct {
		method string
		status int
		args   map[string]any
	}{
		{twprojects.MethodLinkCreate.String(), http.StatusCreated, map[string]any{"project_id": float64(123)}},
		{twprojects.MethodLinkUpdate.String(), http.StatusOK, map[string]any{"id": float64(123)}},
	}
	for _, tool := range tools {
		for _, tt := range tests {
			t.Run(tool.method+"/"+tt.name, func(t *testing.T) {
				mcpServer, body := mcpServerMockWithRequestBody(t, tool.status, []byte(`{"id":"123"}`))
				args := map[string]any{"code": "https://example.com"}
				maps.Copy(args, tool.args)
				if tt.notify != nil {
					args["notify"] = tt.notify
				}
				testutil.ExecuteToolRequest(t, mcpServer, tool.method, args)

				var payload struct {
					Link map[string]json.RawMessage `json:"link"`
				}
				if err := json.Unmarshal(*body, &payload); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if got := string(payload.Link["notify"]); got != tt.want {
					t.Errorf("notify = %s, want %s", got, tt.want)
				}
			})
		}
	}
}
