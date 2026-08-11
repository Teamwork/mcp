package twprojects_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/projects"
)

func TestSearch(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term":             "test",
		"project_id":              float64(123),
		"include_completed_items": true,
		"updated_after":           "2023-01-01T00:00:00Z",
		"extended_search":         true,
		"include":                 []any{"tasks", "projects"},
		"include_highlights":      true,
		"cursor":                  "c858b04ba8b066bcb4f83727c23de6e9238de642",
		"limit":                   float64(10),
	})
}

// TestSearchIncludeHighlights guards that the flag reaches the wire.
func TestSearchIncludeHighlights(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term":        "test",
		"include_highlights": true,
	})

	if got := lastURL.Query().Get("includeHighlights"); got != "true" {
		t.Errorf("expected includeHighlights=%q but got %q", "true", got)
	}
}

// TestSearchOmitsHighlightsByDefault pins that highlights stay opt-in: the
// fragments cost tokens on every hit.
func TestSearchOmitsHighlightsByDefault(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
	})

	if lastURL.Query().Has("includeHighlights") {
		t.Errorf("expected includeHighlights to be unset but got %q", lastURL.Query().Get("includeHighlights"))
	}
}

// TestSearchReturnsHighlightsUntouched guards that the fragments survive the
// truncation pass and output-schema validation.
func TestSearchReturnsHighlightsUntouched(t *testing.T) {
	response := `{
		"search": [{"id": 33, "type": "tasks", "meta": {"highlights": {
			"taskName": ["<em>Test</em> task"],
			"taskDescription": ["something about the <em>test</em>", "second <em>test</em> fragment"]
		}}}],
		"included": {
			"tasks": {"33": {"id": 33, "name": "Test task", "description": "` + strings.Repeat("a", 1234) + `"}}
		}
	}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(response))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term":        "test",
		"include_highlights": true,
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		want := map[string][]string{
			"taskName":        {"<em>Test</em> task"},
			"taskDescription": {"something about the <em>test</em>", "second <em>test</em> fragment"},
		}
		items := searchItemsFromToolResult(t, result)
		if len(items) != 1 {
			t.Fatalf("expected a single search item but got %d", len(items))
		}
		if got := items[0].Highlights(); !reflect.DeepEqual(got, want) {
			t.Errorf("expected highlights %v but got %v", want, got)
		}
	}))
}

// TestSearchDefaultSideloads pins that an unnarrowed search asks for every
// sideload the SDK models.
func TestSearchDefaultSideloads(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
	})

	want := "comments,companies,links,messages,milestones,notebooks,projects,tasklists,tasks,teams,timelogs,users"
	if got := lastURL.Query().Get("include"); got != want {
		t.Errorf("expected include=%q but got %q", want, got)
	}
}

// TestSearchIncludeNarrowsSideloads guards that `include` replaces the
// default sideload list rather than adding to it.
func TestSearchIncludeNarrowsSideloads(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
		"include":     []any{"tasks", "projects"},
	})

	if got := lastURL.Query().Get("include"); got != "tasks,projects" {
		t.Errorf("expected include=%q but got %q", "tasks,projects", got)
	}
}

// TestSearchVerboseDefaultSendsNoSparseFields pins that the default keeps the
// API's full records, leaving size control to truncation.
func TestSearchVerboseDefaultSendsNoSparseFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
	})

	for key := range lastURL.Query() {
		if strings.HasPrefix(key, "fields[") {
			t.Errorf("unexpected sparse-fields parameter %q on a verbose search", key)
		}
	}
}

// TestSearchVerboseFalseRequestsMinimalFields guards that verbose=false asks
// the API for id + label per sideloaded entity instead of full records.
func TestSearchVerboseFalseRequestsMinimalFields(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
		"verbose":     false,
	})

	query := lastURL.Query()
	for key, want := range map[string]string{
		"fields[comments]": "id,title",
		"fields[tasks]":    "id,name",
		"fields[users]":    "id,firstName,lastName",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("expected %s=%q but got %q", key, want, got)
		}
	}
}

// TestSearchIncludeRejectsUnknownType guards that a misspelled entity type
// fails loudly instead of being silently ignored.
func TestSearchIncludeRejectsUnknownType(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
		"include":     []any{"task"},
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if !toolResult.IsError {
			t.Errorf("expected an error for an unknown include value")
		}
	}))
}

// TestSearchTruncatesSideloadContent covers the content cap per sideload
// section (each keeps content under a different attribute) and the marker
// naming the per-entity get tool.
func TestSearchTruncatesSideloadContent(t *testing.T) {
	long := strings.Repeat("a", 1234)

	response := `{
		"search": [{"id": 1, "type": "tasks"}],
		"included": {
			"comments": {"11": {"id": 11, "title": "` + long + `"}},
			"messages": {"22": {"id": 22, "title": "subject", "body": "` + long + `"}},
			"tasks": {"33": {"id": 33, "name": "a task", "description": "` + long + `"}}
		}
	}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(response))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		included := searchIncludedFromToolResult(t, result)
		for _, entry := range []struct {
			section string
			id      string
			field   string
			method  string
		}{
			{"comments", "11", "title", twprojects.MethodCommentGet.String()},
			{"messages", "22", "body", twprojects.MethodMessageGet.String()},
			{"tasks", "33", "description", twprojects.MethodTaskGet.String()},
		} {
			got, ok := included[entry.section][entry.id][entry.field].(string)
			if !ok {
				t.Fatalf("expected a string at included.%s.%s.%s", entry.section, entry.id, entry.field)
			}
			if !strings.HasPrefix(got, strings.Repeat("a", 500)) {
				t.Errorf("expected the first 500 characters of %s.%s to survive, got %q",
					entry.section, entry.field, got)
			}
			if strings.Contains(got, strings.Repeat("a", 501)) {
				t.Errorf("expected %s.%s to stop at 500 characters, got %q",
					entry.section, entry.field, got)
			}
			for _, want := range []string{"truncated", "1,234 chars total", "id=" + entry.id, entry.method} {
				if !strings.Contains(got, want) {
					t.Errorf("expected the %s.%s marker to carry %q but got %q",
						entry.section, entry.field, want, got)
				}
			}
		}

		// A message's title is its subject line, not its content, so it stays.
		if got := included["messages"]["22"]["title"]; got != "subject" {
			t.Errorf("expected the message title untouched but got %q", got)
		}
	}))
}

// TestSearchLeavesShortContentAlone is the control: content within the limit
// comes back untouched.
func TestSearchLeavesShortContentAlone(t *testing.T) {
	// 500 emoji are 2000 bytes: guards that the cap counts runes, not bytes.
	emoji := strings.Repeat("🙂", 500)
	encodedEmoji, err := json.Marshal(emoji)
	if err != nil {
		t.Fatalf("failed to encode body: %v", err)
	}

	response := `{
		"search": [{"id": 1, "type": "tasks"}],
		"included": {
			"comments": {"11": {"id": 11, "title": "short comment"}},
			"tasks": {"33": {"id": 33, "description": ` + string(encodedEmoji) + `}}
		}
	}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(response))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodSearch.String(), map[string]any{
		"search_term": "test",
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
		t.Helper()
		testutil.CheckMessage(t, result)

		included := searchIncludedFromToolResult(t, result)
		if got := included["comments"]["11"]["title"]; got != "short comment" {
			t.Errorf("expected the comment content untouched but got %q", got)
		}
		if got := included["tasks"]["33"]["description"]; got != emoji {
			t.Errorf("expected the task description untouched but got %q", got)
		}
	}))
}

// searchItemsFromToolResult decodes the hit list out of a search tool result,
// as the SDK type so the wire format is covered too.
func searchItemsFromToolResult(t *testing.T, result mcp.Result) []projects.SearchItem {
	t.Helper()

	var payload struct {
		Items []projects.SearchItem `json:"search"`
	}
	if err := json.Unmarshal([]byte(searchTextFromToolResult(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	return payload.Items
}

// searchIncludedFromToolResult decodes the sideload sections out of a search
// tool result.
func searchIncludedFromToolResult(t *testing.T, result mcp.Result) map[string]map[string]map[string]any {
	t.Helper()

	var payload struct {
		Included map[string]map[string]map[string]any `json:"included"`
	}
	if err := json.Unmarshal([]byte(searchTextFromToolResult(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode tool output: %v", err)
	}
	return payload.Included
}

// searchTextFromToolResult returns the raw payload of a search tool result.
func searchTextFromToolResult(t *testing.T, result mcp.Result) string {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	return text.Text
}
