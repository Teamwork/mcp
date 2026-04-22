//nolint:lll
package twspaces_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestCommentGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"comment":{"id":100,"content":"Great docs!","state":"active","isPrivate":false,"page":{"id":10,"type":"page"},"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentGet.String(), map[string]any{
		"spaceId":   float64(1),
		"pageId":    float64(10),
		"commentId": float64(100),
	})
}

func TestCommentList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"comments":[{"id":100,"content":"Great docs!","state":"active","isPrivate":false,"page":{"id":10,"type":"page"},"space":{"id":1,"type":"space"},"replies":[]}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentList.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
	})
}

func TestCommentCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"comment":{"id":100,"content":"Great docs!","state":"active","isPrivate":false,"page":{"id":10,"type":"page"},"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentCreate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"content": "Great docs!",
	})
}

func TestCommentCreateReply(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"comment":{"id":101,"parentId":100,"content":"Thanks!","state":"active","isPrivate":false,"page":{"id":10,"type":"page"},"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentCreate.String(), map[string]any{
		"spaceId":  float64(1),
		"pageId":   float64(10),
		"content":  "Thanks!",
		"parentId": float64(100),
	})
}

func TestCommentUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"comment":{"id":100,"content":"Updated comment","state":"active","isPrivate":false,"page":{"id":10,"type":"page"},"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentUpdate.String(), map[string]any{
		"spaceId":   float64(1),
		"pageId":    float64(10),
		"commentId": float64(100),
		"content":   "Updated comment",
	})
}

func TestCommentDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodCommentDelete.String(), map[string]any{
		"spaceId":   float64(1),
		"pageId":    float64(10),
		"commentId": float64(100),
	})
}
