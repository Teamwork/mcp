package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTasklistBudgetList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"meta":{"page":{"hasMore":false}},"tasklistBudgets":[{"id":98765,"projectBudgetId":12345,"tasklistId":4567}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTasklistBudgetList.String(), map[string]any{
		"project_budget_id": float64(12345),
		"page":              float64(1),
		"page_size":         float64(10),
	})

}

func TestProjectBudgetList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"meta":{"page":{"hasMore":false}},"budgets":[{"id":13579,"projectId":2468}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectBudgetList.String(), map[string]any{
		"project_ids": []float64{2468, 9753},
		"status":      "active",
		"limit":       float64(5),
		"page_size":   float64(2),
		"cursor":      "next-cursor-token",
	})
}
