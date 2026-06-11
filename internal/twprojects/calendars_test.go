package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCalendarCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"calendar":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarCreate.String(), map[string]any{
		"name": "blocked_time",
		"type": "blocked_time",
	})
}

func TestCalendarDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCalendarList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCalendarList.String(), map[string]any{
		"page":      float64(1),
		"page_size": float64(10),
	})
}

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
