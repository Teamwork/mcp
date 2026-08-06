package twprojects_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// TestCommentListTruncatesBody covers the cap on comment bodies and, just as
// importantly, the marker that admits to it. A caller handed half a comment
// with no sign of it reasons over the half confidently, so the assertions below
// pin all three things the marker has to carry: that the text stops early, how
// much text there is in total, and the call that returns the rest.
func TestCommentListTruncatesBody(t *testing.T) {
	const total = 1234
	body := strings.Repeat("a", total)

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"comments":[{"id":13674684,"body":"`+body+`"}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			got := commentBodiesFromToolResult(t, result)
			if len(got) != 1 {
				t.Fatalf("expected one comment but got %d", len(got))
			}
			if !strings.HasPrefix(got[0], strings.Repeat("a", 500)) {
				t.Errorf("expected the first 500 characters to survive, got %q", got[0])
			}
			if strings.Contains(got[0], strings.Repeat("a", 501)) {
				t.Errorf("expected the body to stop at 500 characters, got %q", got[0])
			}
			for _, want := range []string{"truncated", "1,234 chars total", "id=13674684"} {
				if !strings.Contains(got[0], want) {
					t.Errorf("expected the marker to carry %q but got %q", want, got[0])
				}
			}
			if !strings.Contains(got[0], twprojects.MethodCommentGet.String()) {
				t.Errorf("expected the marker to name %s but got %q", twprojects.MethodCommentGet, got[0])
			}
		}))
}

// TestCommentListLeavesShortBodyAlone is the control on the cap. The median
// comment is under it, so a truncation that fired on every record would be
// paying the marker's cost on the majority of a response for nothing.
func TestCommentListLeavesShortBodyAlone(t *testing.T) {
	// Multi-byte characters make this a rune count rather than a byte count:
	// 500 emoji are 2000 bytes, and cutting by bytes would both truncate a body
	// that is within the cap and split a character while doing it.
	for name, body := range map[string]string{
		"short":    "Looks good to me",
		"at limit": strings.Repeat("a", 500),
		"emoji":    strings.Repeat("🙂", 500),
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("failed to encode body: %v", err)
			}

			mcpServer := mcpServerMock(t, http.StatusOK,
				[]byte(`{"comments":[{"id":1,"body":`+string(encoded)+`}]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{},
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					testutil.CheckMessage(t, result)

					got := commentBodiesFromToolResult(t, result)
					if len(got) != 1 {
						t.Fatalf("expected one comment but got %d", len(got))
					}
					if got[0] != body {
						t.Errorf("expected the body untouched but got %q", got[0])
					}
				}))
		})
	}
}

// TestCommentListTruncatesSelectedBody guards that naming body in `fields` does
// not opt out of the cap: an explicit selection of body across a few hundred
// records is exactly the payload the cap exists to bound, and get_comment
// returns full text either way.
func TestCommentListTruncatesSelectedBody(t *testing.T) {
	body := strings.Repeat("a", 1234)

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"comments":[{"id":7,"body":"`+body+`"}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentList.String(), map[string]any{
		"fields": []any{string(projects.CommentFieldBody)},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		got := commentBodiesFromToolResult(t, result)
		if len(got) != 1 {
			t.Fatalf("expected one comment but got %d", len(got))
		}
		if !strings.Contains(got[0], "truncated") {
			t.Errorf("expected a selected body to be truncated too, got %q", got[0])
		}
	}))
}

// commentBodiesFromToolResult decodes the comment bodies out of a list_comments
// tool result.
func commentBodiesFromToolResult(t *testing.T, result mcp.Result) []string {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	var payload struct {
		Comments []struct {
			Body string `json:"body"`
		} `json:"comments"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	bodies := make([]string, len(payload.Comments))
	for i, comment := range payload.Comments {
		bodies[i] = comment.Body
	}
	return bodies
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
