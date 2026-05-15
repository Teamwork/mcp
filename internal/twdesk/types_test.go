//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestTypeCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"ticket_type":{"id":123,"name":"Bug Report"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeCreate.String(), map[string]any{
		"name":                    "Bug Report",
		"displayOrder":            nil,
		"enabledForFutureInboxes": nil,
	})
}

func TestTypeUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_type":{"id":123,"name":"Feature Request"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeUpdate.String(), map[string]any{
		"id":                      float64(123),
		"name":                    "Feature Request",
		"displayOrder":            nil,
		"enabledForFutureInboxes": nil,
	})
}

func TestTypeGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_type":{"id":123,"name":"Support"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeGet.String(), map[string]any{
		"id":     float64(123),
		"fields": nil,
	})
}

func TestTypeList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_types":[{"id":123,"name":"Bug Report"},{"id":124,"name":"Feature Request"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeList.String(), map[string]any{
		"name":           []string{"Bug Report", "Feature Request"},
		"inboxIDs":       nil,
		"page":           float64(1),
		"pageSize":       float64(10),
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestTypeListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"ticket_types":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTypeList.String(), map[string]any{
		"name":           nil,
		"inboxIDs":       nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}
