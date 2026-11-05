package twprojects_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTimerCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timer":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerCreate.String(), map[string]any{
		"description":         "Example timer description",
		"billable":            true,
		"running":             true,
		"seconds":             float64(3600), // 1 hour
		"stop_running_timers": true,
		"project_id":          float64(123),
		"task_id":             float64(456),
	})
}

func TestTimerUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerUpdate.String(), map[string]any{
		"id":          float64(123),
		"description": "Example timer description",
		"billable":    true,
		"running":     true,
		"project_id":  float64(123),
		"task_id":     float64(456),
	})
}

func TestTimerPause(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerPause.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerResume(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerResume.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerComplete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerComplete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTimerList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTimerList.String(), map[string]any{
		"user_id":             float64(123),
		"task_id":             float64(456),
		"project_id":          float64(789),
		"running_timers_only": true,
		"page":                float64(1),
		"page_size":           float64(10),
	})
}

// TestTimerResultsCarryNoWebLink pins that timers get no web link: the web app
// has no page for a timer, so any link lands on the home page.
func TestTimerResultsCarryNoWebLink(t *testing.T) {
	tests := []struct {
		name string
		tool string
		body string
		args map[string]any
	}{
		{"get", twprojects.MethodTimerGet.String(), `{"timer":{"id":123}}`, map[string]any{"id": float64(123)}},
		{"get with fields", twprojects.MethodTimerGet.String(), `{"timer":{"id":123}}`,
			map[string]any{"id": float64(123), "fields": []any{"description"}}},
		{"list", twprojects.MethodTimerList.String(), `{"timers":[{"id":123}]}`, map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer := withCustomerURL(mcpServerMock(t, http.StatusOK, []byte(tt.body)))
			testutil.ExecuteToolRequest(t, mcpServer, tt.tool, tt.args,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					testutil.CheckMessage(t, result)

					toolResult, ok := result.(*mcp.CallToolResult)
					if !ok {
						t.Fatalf("unexpected result type %T", result)
					}
					for _, content := range toolResult.Content {
						if text, ok := content.(*mcp.TextContent); ok && strings.Contains(text.Text, "webLink") {
							t.Errorf("expected no web link but got %s", text.Text)
						}
					}
					structured, err := json.Marshal(toolResult.StructuredContent)
					if err != nil {
						t.Fatalf("failed to encode structured content: %v", err)
					}
					if strings.Contains(string(structured), "webLink") {
						t.Errorf("expected no web link in structured content but got %s", structured)
					}
				}))
		})
	}
}
