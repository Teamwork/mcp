package twprojects_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/projects"
)

func TestCommentCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreate.String(), map[string]any{
		"object": map[string]any{
			"type": "tasks",
			"id":   float64(123),
		},
		"body":                "Example",
		"content_type":        "TEXT",
		"notify_current_user": true,
		"notify": map[string]any{
			"user_ids":    []any{float64(1), float64(2)},
			"team_ids":    []any{float64(3)},
			"company_ids": []any{float64(4)},
		},
	})
}

func TestCommentUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentUpdate.String(), map[string]any{
		"id":                  float64(123),
		"body":                "Example",
		"content_type":        "TEXT",
		"notify_current_user": true,
		"notify":              "all",
	})
}

func TestCommentDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCommentList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

// TestCommentListDropsHTMLBody pins htmlBody out of the default list selection.
// It is the same content as body a second time and the larger of the two, so a
// list response that carries both spends most of its payload on a duplicate.
// The endpoint implements sparse fieldsets, so this is asserted on the outgoing
// query: the mock replies with the same canned body whatever is asked for, and
// a selection that never reaches the API means the response quietly carries
// htmlBody anyway.
func TestCommentListDropsHTMLBody(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{})

	selections := sparseFieldsParams(lastURL.Query())
	if len(selections) != 1 {
		t.Fatalf("expected exactly one fields[...] selection but got %v (raw query: %s)",
			selections, lastURL.RawQuery)
	}
	for entity, got := range selections {
		selected := strings.Split(got, ",")
		if slices.Contains(selected, string(projects.CommentFieldHTMLBody)) {
			t.Errorf("expected htmlBody out of fields[%s] but got %q", entity, got)
		}
		// Everything else the SDK models stays in, so the cut is the duplicate
		// body rather than a narrowing of the record.
		for _, attribute := range helpers.SparseFieldNames[projects.CommentField, projects.Comment]() {
			if attribute == projects.CommentFieldHTMLBody {
				continue
			}
			if !slices.Contains(selected, string(attribute)) {
				t.Errorf("expected %s in fields[%s] but got %q", attribute, entity, got)
			}
		}
	}
}

// TestCommentListHTMLBodyRemainsSelectable guards the escape hatch: dropping
// htmlBody is a default, not a removal, so a caller that names it still gets it.
func TestCommentListHTMLBodyRemainsSelectable(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"fields": []any{string(projects.CommentFieldHTMLBody)},
	})

	for entity, got := range sparseFieldsParams(lastURL.Query()) {
		if !slices.Contains(strings.Split(got, ","), string(projects.CommentFieldHTMLBody)) {
			t.Errorf("expected htmlBody in fields[%s] but got %q", entity, got)
		}
	}
}

func TestCommentListByFileVersion(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":     "test",
		"file_version_id": float64(123),
		"updated_after":   "2025-01-01T00:00:00Z",
		"page":            float64(1),
		"page_size":       float64(10),
	})
}

func TestCommentListByMilestone(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"milestone_id":  float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByNotebook(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"notebook_id":   float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByTask(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"task_id":       float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}

func TestCommentListByLink(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"search_term":   "test",
		"link_id":       float64(123),
		"updated_after": "2025-01-01T00:00:00Z",
		"page":          float64(1),
		"page_size":     float64(10),
	})
}
