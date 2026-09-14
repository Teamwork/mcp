package twspaces

import (
	"fmt"
	"strings"

	"github.com/teamwork/mcp/pkg/helpers"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
)

// Bounds on the number of page nodes twspaces-list_pages returns.
//
// GET /spaces/api/v1/spaces/{spaceId}/pages.json answers with a space's whole
// page tree and reads neither pageSize nor pageOffset, so the cap has to be
// applied here or there is no cap at all. Left unbounded this is the largest
// response the server produces — a space with thousands of pages answers in
// hundreds of kilobytes, and a few such calls fill a model's context window on
// their own, which costs the caller the turn rather than just the tool result.
const (
	defaultPageTreeLimit = 100
	maxPageTreeLimit     = 500
)

// pageListResponse is the SDK's list response plus the marker the handler sets
// when it drops nodes. The embedded pointer keeps the API's own fields at the
// top level, so a capped response is shaped exactly like an uncapped one.
type pageListResponse struct {
	*spacesmodels.PagesResponse
	Truncated string `json:"truncated,omitempty"`
}

// pageTreeLimit resolves the caller's pageSize into a node cap. A caller that
// names none gets the default rather than the whole tree, since the API's idea
// of "no limit" is what makes this tool unsafe to call blind.
func pageTreeLimit(pageSize int) int {
	switch {
	case pageSize <= 0:
		return defaultPageTreeLimit
	case pageSize > maxPageTreeLimit:
		return maxPageTreeLimit
	default:
		return pageSize
	}
}

// capPageList bounds a page tree to the window [offset, offset+limit) in
// depth-first order, returning the response to send. A tree that already fits
// is returned untouched: the marker costs nothing to add but is noise on the
// spaces that never had the problem.
func capPageList(pages *spacesmodels.PagesResponse, offset, limit int) any {
	if pages == nil {
		return pages
	}
	if offset < 0 {
		offset = 0
	}
	total := countPages(pages.Pages.ChildPages)
	if offset == 0 && total <= limit {
		return pages
	}

	capped := *pages
	selected, kept := selectPages(pages.Pages.ChildPages, offset, limit)
	capped.Pages.ChildPages = selected

	return pageListResponse{
		PagesResponse: &capped,
		Truncated:     pageTreeMarker(total, offset, kept),
	}
}

// countPages counts every node in the forest, at every depth.
func countPages(nodes []spacesmodels.PageTreeNode) int {
	var count int
	for _, node := range nodes {
		count += 1 + countPages(node.ChildPages)
	}
	return count
}

// selectPages keeps the pages whose depth-first position falls in
// [offset, offset+limit), reporting how many that came to. A kept page whose
// parent fell outside the window is re-attached to the nearest ancestor that is
// inside it, so the caller still receives a tree rather than a broken one.
func selectPages(nodes []spacesmodels.PageTreeNode, offset, limit int) ([]spacesmodels.PageTreeNode, int) {
	var position, kept int

	var walk func([]spacesmodels.PageTreeNode) []spacesmodels.PageTreeNode
	walk = func(nodes []spacesmodels.PageTreeNode) []spacesmodels.PageTreeNode {
		selected := []spacesmodels.PageTreeNode{}
		for _, node := range nodes {
			// Decided before descending so the budget is spent in the same
			// depth-first order the positions are numbered in.
			include := position >= offset && kept < limit
			position++
			if include {
				kept++
			}

			children := walk(node.ChildPages)
			if include {
				node.ChildPages = children
				selected = append(selected, node)
			} else {
				selected = append(selected, children...)
			}
		}
		return selected
	}

	return walk(nodes), kept
}

// pageTreeMarker spells out what was dropped, in the style of twprojects'
// truncateContent: that the data stops early, how much there is in total, and
// the calls that reach the rest.
func pageTreeMarker(total, offset, shown int) string {
	var marker strings.Builder
	fmt.Fprintf(&marker, "...[truncated — %s pages total, showing %s from offset %s",
		helpers.FormatThousands(total), helpers.FormatThousands(shown), helpers.FormatThousands(offset))
	if next := offset + shown; next < total {
		fmt.Fprintf(&marker, ", %s(pageOffset=%d) for the next set", MethodPageList, next)
	}
	fmt.Fprintf(&marker, ", %s(pageId=…) for one page in full]", MethodPageGet)
	return marker.String()
}
