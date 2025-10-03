package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestUsersWorkload(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUsersWorkload.String(), map[string]any{
		"start_date":       "2023-01-01",
		"end_date":         "2023-01-31",
		"user_ids":         []float64{1, 2, 3},
		"user_company_ids": []float64{4, 5, 6},
		"user_team_ids":    []float64{7, 8, 9},
		"project_ids":      []float64{10, 11, 12},
		"page":             float64(1),
		"page_size":        float64(10),
	})
}
