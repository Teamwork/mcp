package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCalendarEventList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarEventList.String(), map[string]any{
		"calendar_id":        float64(123),
		"started_after_date": "2023-01-01",
		"ended_before_date":  "2023-12-31",
		"limit":              float64(25),
		"cursor":             "abc123",
	})
}
