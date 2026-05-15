//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestTagCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"tag":{"id":123,"name":"urgent","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagCreate.String(), map[string]any{
		"name":  "urgent",
		"color": "red",
	})
}

func TestTagUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tag":{"id":123,"name":"important","color":"orange"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagUpdate.String(), map[string]any{
		"id":    float64(123),
		"name":  "important",
		"color": "orange",
	})
}

func TestTagGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tag":{"id":123,"name":"urgent","color":"red"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagGet.String(), map[string]any{
		"id":     float64(123),
		"fields": nil,
	})
}

func TestTagList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[{"id":123,"name":"urgent","color":"red"},{"id":124,"name":"important","color":"orange"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagList.String(), map[string]any{
		"name":           "urgent",
		"color":          "red",
		"inboxIDs":       nil,
		"page":           float64(1),
		"pageSize":       float64(10),
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}

func TestTagListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"tags":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodTagList.String(), map[string]any{
		"name":           nil,
		"color":          nil,
		"inboxIDs":       nil,
		"page":           nil,
		"pageSize":       nil,
		"orderBy":        nil,
		"orderDirection": nil,
		"fields":         nil,
	})
}
