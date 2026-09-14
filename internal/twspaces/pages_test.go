//nolint:lll
package twspaces_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twspaces"
)

func TestPageGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Getting Started","slug":"getting-started","content":"<p>Welcome</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageGet.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
	})
}

func TestPageList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"pages":{"id":0,"slug":"","title":"root","childPages":[{"id":10,"slug":"getting-started","title":"Getting Started","childPages":[]}]}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId": float64(1),
	})
}

func TestPageHome(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":1,"title":"Home","slug":"home","content":"<p>Welcome</p>","isHomePage":true,"state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageHome.String(), map[string]any{
		"spaceId": float64(1),
	})
}

func TestPageCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"New Page","slug":"new-page","content":"<p>Hello</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageCreate.String(), map[string]any{
		"spaceId": float64(1),
		"title":   "New Page",
		"content": "<p>Hello</p>",
	})
}

func TestPageCreateWithOptionals(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"New Page","slug":"new-page","content":"<p>Hello</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageCreate.String(), map[string]any{
		"spaceId":   float64(1),
		"title":     "New Page",
		"content":   "<p>Hello</p>",
		"parentId":  float64(5),
		"slug":      "new-page",
		"isPublish": true,
	})
}

func TestPageDuplicate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":11,"title":"Copy of Getting Started","slug":"copy-of-getting-started","content":"<p>Welcome</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageDuplicate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"title":   "Copy of Getting Started",
	})
}

func TestPageUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Updated Page","slug":"getting-started","content":"<p>Updated</p>","state":"active","space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageUpdate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"title":   "Updated Page",
		"content": "<p>Updated</p>",
	})
}

// TestPageUpdateWarnsWhenActiveDraft verifies that updating a page that already
// has an active editor draft (draftVersion > 1) surfaces the draft-sync warning,
// since the API write updates only the published content and not the draft.
func TestPageUpdateWarnsWhenActiveDraft(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Updated Page","slug":"getting-started","content":"<p>Updated</p>","state":"active","draftVersion":50,"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageUpdate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"content": "<p>Updated</p>",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		assertDraftWarning(t, result, true)
	}))
}

// TestPageUpdateNoWarnWhenNoActiveDraft verifies that updating a page whose
// draftVersion is <= 1 (no real editor draft yet) does not surface the warning,
// since the editor will seed the draft from the published content on first edit.
func TestPageUpdateNoWarnWhenNoActiveDraft(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"page":{"id":10,"title":"Updated Page","slug":"getting-started","content":"<p>Updated</p>","state":"active","draftVersion":1,"space":{"id":1,"type":"space"}}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageUpdate.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
		"content": "<p>Updated</p>",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		assertDraftWarning(t, result, false)
	}))
}

func assertDraftWarning(t *testing.T, result mcp.Result, want bool) {
	t.Helper()
	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if toolResult.IsError {
		t.Fatalf("tool returned an error result: %v", toolResult.Content)
	}
	var found bool
	for _, content := range toolResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok &&
			strings.Contains(textContent.Text, "Draft-sync warning") {
			found = true
		}
	}
	if found != want {
		t.Errorf("draft-sync warning present = %v, want %v", found, want)
	}
}

func TestPageDelete(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusNoContent, []byte(``))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageDelete.String(), map[string]any{
		"spaceId": float64(1),
		"pageId":  float64(10),
	})
}

// pageTree builds a list response whose root carries `count` pages: a flat run
// of ids 1..count, except that every third page is nested one level under the
// page before it, so the depth-first order the cap walks is not the same as the
// order the JSON nests them in.
func pageTree(count int) []byte {
	type node struct {
		ID         int64   `json:"id"`
		Slug       string  `json:"slug"`
		Title      string  `json:"title"`
		ChildPages []*node `json:"childPages"`
	}

	root := &node{Title: "root", ChildPages: []*node{}}
	var previous *node
	for id := int64(1); id <= int64(count); id++ {
		page := &node{
			ID:         id,
			Slug:       fmt.Sprintf("page-%d", id),
			Title:      fmt.Sprintf("Page %d", id),
			ChildPages: []*node{},
		}
		if previous != nil && id%3 == 0 {
			previous.ChildPages = append(previous.ChildPages, page)
		} else {
			root.ChildPages = append(root.ChildPages, page)
		}
		previous = page
	}

	encoded, err := json.Marshal(map[string]any{"pages": root})
	if err != nil {
		panic(err)
	}
	return encoded
}

// pageListResult decodes a list_pages result into the page ids it carries, in
// depth-first order, and the truncation marker it did or did not come with.
func pageListResult(t *testing.T, result mcp.Result) ([]int64, string) {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}

	type node struct {
		ID         int64  `json:"id"`
		ChildPages []node `json:"childPages"`
	}
	var payload struct {
		Pages     node   `json:"pages"`
		Truncated string `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}

	var ids []int64
	var walk func([]node)
	walk = func(nodes []node) {
		for _, n := range nodes {
			ids = append(ids, n.ID)
			walk(n.ChildPages)
		}
	}
	walk(payload.Pages.ChildPages)
	return ids, payload.Truncated
}

// TestPageListLeavesSmallTreeAlone is the control on the cap. Most spaces are
// well under it, and a marker on those would be noise on a response that left
// nothing out.
func TestPageListLeavesSmallTreeAlone(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, pageTree(7))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId": float64(1),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		ids, marker := pageListResult(t, result)
		if len(ids) != 7 {
			t.Errorf("expected all 7 pages but got %d: %v", len(ids), ids)
		}
		if marker != "" {
			t.Errorf("expected no truncation marker but got %q", marker)
		}
	}))
}

// TestPageListCapsTree covers the cap and, just as importantly, the marker that
// admits to it: a caller handed part of a space's page tree with no sign of it
// reads the missing pages as pages that do not exist.
func TestPageListCapsTree(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, pageTree(10))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":  float64(1),
		"pageSize": float64(4),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		ids, marker := pageListResult(t, result)
		if want := []int64{1, 2, 3, 4}; !slices.Equal(ids, want) {
			t.Errorf("expected the first %v in depth-first order but got %v", want, ids)
		}
		for _, want := range []string{
			"truncated", "10 pages total", "showing 4 from offset 0",
			twspaces.MethodPageList.String() + "(pageOffset=4)",
			twspaces.MethodPageGet.String(),
		} {
			if !strings.Contains(marker, want) {
				t.Errorf("expected the marker to carry %q but got %q", want, marker)
			}
		}
	}))
}

// TestPageListDefaultsTheCap guards that a caller naming no pageSize is still
// bounded: the endpoint reads neither pagination parameter, so an unbounded
// default would leave the whole tree on the wire for every caller that does not
// know to ask for less.
func TestPageListDefaultsTheCap(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, pageTree(150))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId": float64(1),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		ids, marker := pageListResult(t, result)
		if len(ids) != 100 {
			t.Errorf("expected the default cap of 100 pages but got %d", len(ids))
		}
		if !strings.Contains(marker, "150 pages total") {
			t.Errorf("expected the marker to report the whole tree but got %q", marker)
		}
	}))
}

// TestPageListPageOffsetReturnsTheNextSet walks the second call the marker tells
// the caller to make, and checks the marker stops offering a third once the
// window reaches the end of the tree.
func TestPageListPageOffsetReturnsTheNextSet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, pageTree(10))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":    float64(1),
		"pageSize":   float64(6),
		"pageOffset": float64(6),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		ids, marker := pageListResult(t, result)
		if want := []int64{7, 8, 9, 10}; !slices.Equal(ids, want) {
			t.Errorf("expected the remaining %v but got %v", want, ids)
		}
		if !strings.Contains(marker, "showing 4 from offset 6") {
			t.Errorf("expected the marker to report the window but got %q", marker)
		}
		if strings.Contains(marker, "pageOffset=10") {
			t.Errorf("expected no next set past the end of the tree but got %q", marker)
		}
	}))
}

// TestPageListKeepsTheTreeWhole pins the reparenting the cap has to do: page 3
// is nested under page 2, so a window starting at page 3 leaves a kept page
// whose parent is outside it. Dropping such a page would lose it from both
// calls, and leaving it nested under a page that is no longer there is not
// something the response can express.
func TestPageListKeepsTheTreeWhole(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, pageTree(10))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":    float64(1),
		"pageSize":   float64(2),
		"pageOffset": float64(2),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		ids, _ := pageListResult(t, result)
		if want := []int64{3, 4}; !slices.Equal(ids, want) {
			t.Errorf("expected the nested page to survive its parent being skipped, got %v", ids)
		}
	}))
}
