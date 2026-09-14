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

// spaceContentTree builds a v2 list response: an open tree of `open` pages with
// ids 1..open, and a private forest of `private` pages numbered on from there.
// Every third page in each tree is nested one level under the page before it,
// so the depth-first order the cap walks is not the order the JSON nests them
// in.
func spaceContentTree(open, private int) []byte {
	type node struct {
		ID         int64   `json:"id"`
		Slug       string  `json:"slug"`
		Title      string  `json:"title"`
		ChildPages []*node `json:"childPages"`
	}

	build := func(first, count int64) []*node {
		roots := []*node{}
		var previous *node
		for id := first; id < first+count; id++ {
			page := &node{
				ID:         id,
				Slug:       fmt.Sprintf("page-%d", id),
				Title:      fmt.Sprintf("Page %d", id),
				ChildPages: []*node{},
			}
			if previous != nil && id%3 == 0 {
				previous.ChildPages = append(previous.ChildPages, page)
			} else {
				roots = append(roots, page)
			}
			previous = page
		}
		return roots
	}

	encoded, err := json.Marshal(map[string]any{
		"spaceContent": map[string]any{
			"pages":   &node{Title: "root", ChildPages: build(1, int64(open))},
			"private": build(int64(open)+1, int64(private)),
		},
	})
	if err != nil {
		panic(err)
	}
	return encoded
}

// pageTreeResult decodes a list_pages result that carries both trees into the
// page ids of each, in depth-first order, and the truncation marker.
func pageTreeResult(t *testing.T, result mcp.Result) (open, private []int64, marker string) {
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
		Private   []node `json:"private"`
		Truncated string `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}

	var walk func([]node, *[]int64)
	walk = func(nodes []node, into *[]int64) {
		for _, n := range nodes {
			*into = append(*into, n.ID)
			walk(n.ChildPages, into)
		}
	}
	walk(payload.Pages.ChildPages, &open)
	walk(payload.Private, &private)
	return open, private, payload.Truncated
}

// TestPageListIncludePrivateReachesTheWire pins which route each call takes.
// The two versions serve the same path under a different version segment and
// answer the same query, so a call that reaches the wrong one comes back with a
// well-formed response that is simply missing the private tree — which reads as
// a space with no private pages rather than as the wrong request.
func TestPageListIncludePrivateReachesTheWire(t *testing.T) {
	for _, tc := range []struct {
		name           string
		includePrivate any
		wantPath       string
	}{
		{"omitted", nil, "/spaces/api/v1/spaces/1/pages.json"},
		{"false", false, "/spaces/api/v1/spaces/1/pages.json"},
		{"true", true, "/spaces/api/v2/spaces/1/pages.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mcpServer, lastRequestURL, cleanup := testutil.SpacesMCPServerMockWithRequestURL(t,
				http.StatusOK, spaceContentTree(2, 1))
			defer cleanup()

			arguments := map[string]any{"spaceId": float64(1)}
			if tc.includePrivate != nil {
				arguments["includePrivate"] = tc.includePrivate
			}

			testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), arguments)

			if got := lastRequestURL().Path; got != tc.wantPath {
				t.Errorf("expected the request to reach %s but got %s", tc.wantPath, got)
			}
		})
	}
}

// TestPageListReturnsBothTreesUncapped is the control on the two-tree shape:
// under the cap, both trees arrive whole, beside each other, with no marker.
// Merging them would answer the question the caller asked includePrivate to
// distinguish — which of these pages are restricted — with a tree that cannot
// say.
func TestPageListReturnsBothTreesUncapped(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, spaceContentTree(4, 3))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":        float64(1),
		"includePrivate": true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		open, private, marker := pageTreeResult(t, result)
		if want := []int64{1, 2, 3, 4}; !slices.Equal(open, want) {
			t.Errorf("expected the open tree %v but got %v", want, open)
		}
		if want := []int64{5, 6, 7}; !slices.Equal(private, want) {
			t.Errorf("expected the private tree %v but got %v", want, private)
		}
		if marker != "" {
			t.Errorf("expected no truncation marker but got %q", marker)
		}
	}))
}

// TestPageListCapsAcrossBothTrees is the one the second tree could have walked
// straight past: a cap that counted and windowed each tree on its own would
// answer twice the pageSize asked for, which is the unbounded response the cap
// exists to stop. The marker's total has to span both for the same reason — a
// total naming only the open tree understates what is left to read.
func TestPageListCapsAcrossBothTrees(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, spaceContentTree(4, 4))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":        float64(1),
		"includePrivate": true,
		"pageSize":       float64(6),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		open, private, marker := pageTreeResult(t, result)
		if want := []int64{1, 2, 3, 4}; !slices.Equal(open, want) {
			t.Errorf("expected the whole open tree %v but got %v", want, open)
		}
		// The budget is spent on the open tree first, so the private tree gets
		// what is left of it rather than a fresh six.
		if want := []int64{5, 6}; !slices.Equal(private, want) {
			t.Errorf("expected the remaining budget to reach %v but got %v", want, private)
		}
		for _, want := range []string{
			"8 pages total", "showing 6 from offset 0",
			twspaces.MethodPageList.String() + "(pageOffset=6)",
		} {
			if !strings.Contains(marker, want) {
				t.Errorf("expected the marker to carry %q but got %q", want, marker)
			}
		}
	}))
}

// TestPageListWindowStartsInsideThePrivateTree walks the second call the marker
// offers when the first one ended mid-way through the trees. The offset is one
// cursor over both, so a window past the end of the open tree opens inside the
// private one and the open tree comes back empty rather than restarting.
func TestPageListWindowStartsInsideThePrivateTree(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, spaceContentTree(4, 4))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twspaces.MethodPageList.String(), map[string]any{
		"spaceId":        float64(1),
		"includePrivate": true,
		"pageSize":       float64(6),
		"pageOffset":     float64(6),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		open, private, marker := pageTreeResult(t, result)
		if len(open) != 0 {
			t.Errorf("expected the open tree to be behind the window but got %v", open)
		}
		if want := []int64{7, 8}; !slices.Equal(private, want) {
			t.Errorf("expected the rest of the private tree %v but got %v", want, private)
		}
		if !strings.Contains(marker, "showing 2 from offset 6") {
			t.Errorf("expected the marker to report the window but got %q", marker)
		}
		if strings.Contains(marker, "pageOffset=8") {
			t.Errorf("expected no next set past the end of the trees but got %q", marker)
		}
	}))
}
