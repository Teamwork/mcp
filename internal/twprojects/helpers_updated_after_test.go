package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestUpdatedAfterRejectsBeforeEpoch pins the 2007-10-01 floor the v3 endpoints
// apply to updatedAfter: the schema pattern rejects an earlier date, and the
// handler rejects an offset that lands before it in UTC.
func TestUpdatedAfterRejectsBeforeEpoch(t *testing.T) {
	tools := map[string]map[string]any{
		twprojects.MethodTaskList.String():                {},
		twprojects.MethodFileList.String():                {},
		twprojects.MethodProjectStatusUpdateList.String(): {},
		twprojects.MethodSearch.String():                  {"search_term": "roadmap"},
		twprojects.MethodCommentList.String():             {},
		twprojects.MethodProjectList.String():             {},
		twprojects.MethodAllocationList.String():          {},
	}
	tests := []struct {
		value     string
		wantError bool
	}{
		{value: "2007-10-01", wantError: false},
		{value: "2007-10-01T00:00:00Z", wantError: false},
		{value: "2026-08-03T14:30:00Z", wantError: false},
		{value: "2007-09-30", wantError: true},
		{value: "1970-01-01T00:00:00Z", wantError: true},
		{value: "2007-10-01T01:00:00+02:00", wantError: true},
	}
	for tool, args := range tools {
		for _, tt := range tests {
			t.Run(tool+"/"+tt.value, func(t *testing.T) {
				mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
				arguments := map[string]any{"updated_after": tt.value}
				for k, v := range args {
					arguments[k] = v
				}
				testutil.ExecuteToolRequest(t, mcpServer, tool, arguments,
					testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
						toolResult, ok := result.(*mcp.CallToolResult)
						if !ok {
							t.Fatalf("unexpected result type: %T", result)
						}
						if toolResult.IsError != tt.wantError {
							t.Errorf("IsError = %t, want %t (content %v)", toolResult.IsError, tt.wantError, toolResult.Content)
						}
					}))
				if sent := lastURL.Query().Get("updatedAfter"); tt.wantError && sent != "" {
					t.Errorf("expected no request, got updatedAfter=%q", sent)
				} else if !tt.wantError && sent == "" {
					t.Errorf("expected updatedAfter on the wire (raw query: %s)", lastURL.RawQuery)
				}
			})
		}
	}
}
