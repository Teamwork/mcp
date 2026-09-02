package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestProjectStatusUpdateList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectStatusUpdateList.String(), nil)
}

// TestProjectStatusUpdateListFiltersReachTheWire asserts the query string rather
// than the response: the mock replies with the same canned body whatever the
// request carries, so a dropped filter is indistinguishable from a working one.
func TestProjectStatusUpdateListFiltersReachTheWire(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectStatusUpdateList.String(), map[string]any{
		"project_ids":      []any{float64(777), float64(12345)},
		"project_healths":  []any{"bad", "not_set"},
		"active_only":      false,
		"show_deleted":     true,
		"include_archived": true,
		"created_after":    "2026-08-03",
		"updated_after":    "2026-08-10T14:30:00Z",
	})

	query := lastURL.Query()
	for parameter, want := range map[string]string{
		"projectIds":              "777,12345",
		"projectHealths":          "1,0",
		"activeOnly":              "false",
		"showDeleted":             "true",
		"includeArchivedProjects": "true",
		"createdAfter":            "2026-08-03T00:00:00Z",
		"updatedAfter":            "2026-08-10T14:30:00Z",
	} {
		if got := query.Get(parameter); got != want {
			t.Errorf("expected %s=%s but got %q (raw query: %s)", parameter, want, got, lastURL.RawQuery)
		}
	}
}

// TestProjectStatusUpdateListDefaultsToNewestFirst pins the one ordering this
// server does not leave to the endpoint. The endpoint's reference documents an
// ascending default, so an omitted order has to reach the wire as date/desc.
func TestProjectStatusUpdateListDefaultsToNewestFirst(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectStatusUpdateList.String(), nil)

	query := lastURL.Query()
	if got := query.Get("orderBy"); got != "date" {
		t.Errorf("expected orderBy=date but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
	if got := query.Get("orderMode"); got != "desc" {
		t.Errorf("expected orderMode=desc but got %q (raw query: %s)", got, lastURL.RawQuery)
	}
}

// TestProjectStatusUpdateListReturnsTheWholeText guards the decision not to
// truncate. No route reads an update by its own ID and no filter selects one, so
// a truncation marker would name a call that cannot serve the row.
func TestProjectStatusUpdateListReturnsTheWholeText(t *testing.T) {
	text := strings.Repeat("a", 973)
	body := `{"projectUpdates":[{"id":777,"projectId":12345,"text":"` + text + `"}],` +
		`"meta":{"page":{"count":1,"hasMore":false}}}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(body))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectStatusUpdateList.String(), nil,
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			testutil.CheckMessage(t, result)

			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			content, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("unexpected content type: %T", toolResult.Content[0])
			}
			if !strings.Contains(content.Text, text) {
				t.Errorf("the update text was not returned whole: %s", content.Text)
			}
		}),
	)
}

// TestProjectStatusUpdateListValidatesAgainstOutputSchema drives the set and the
// unset date through the published schema. The other tests reply `{}`, which
// leaves every date nil and exercises neither case, and nothing in the server or
// the harness validates structured content — a mismatch only surfaces at a
// validating client, which then discards a response the server returned.
func TestProjectStatusUpdateListValidatesAgainstOutputSchema(t *testing.T) {
	tests := []struct {
		name   string
		update string
	}{{
		name: "live update",
		update: `{"id":777,"projectId":12345,"text":"All good","health":3,"healthLabel":"Good",` +
			`"color":"#8BC34A","createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-02T03:04:05Z",` +
			`"isActive":true,"deleted":false,"deletedAt":null,"deletedBy":null}`,
	}, {
		name: "deleted update",
		update: `{"id":778,"projectId":12345,"text":"Oops","health":0,"healthLabel":"Not Set","color":"",` +
			`"createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-03T03:04:05Z","isActive":false,` +
			`"deleted":true,"deletedAt":"2026-01-04T03:04:05Z","deletedBy":98765}`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"projectUpdates":[` + tt.update + `],"meta":{"page":{"count":1,"hasMore":false}}}`

			mcpServer := mcpServerMock(t, http.StatusOK, []byte(body))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectStatusUpdateList.String(), nil,
				testutil.ExecuteToolRequestWithCheckMessage(
					checkStructuredContentMatchesOutputSchema(
						twprojects.MethodProjectStatusUpdateList.String()),
				))
		})
	}
}
