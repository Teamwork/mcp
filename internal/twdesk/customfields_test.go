package twdesk_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// TestCustomFieldListRequest pins the route, the options sideload a dropdown or
// checkboxes condition is written in, and the inbox filter.
func TestCustomFieldListRequest(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"customfields":[{"id":12,"agentLabel":"Plan","kind":"dropdown",`+
			`"options":[{"id":14,"type":"customfieldoptions"}]}],`+
			`"included":{"customfieldoptions":[{"id":14,"name":"Gold"}]}}`))
	defer cleanup()

	text := lookupResultText(t, mcpServer, twdesk.MethodCustomFieldList, map[string]any{
		"inboxIDs": []float64{1, 2},
		"page":     nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil, "fields": nil,
	})

	requestURL := lastRequestURL()
	if got, want := requestURL.Path, "/desk/api/v2/customfields.json"; got != want {
		t.Errorf("unexpected request path: got %q, want %q", got, want)
	}
	query := requestURL.Query()
	if got, want := query.Get("includes"), "customfieldoptions"; got != want {
		t.Errorf("query parameter \"includes\": got %q, want %q", got, want)
	}
	var filter map[string]map[string][]int
	if err := json.Unmarshal([]byte(query.Get("filter")), &filter); err != nil {
		t.Fatalf("filter should be a JSON document, got %q: %v", query.Get("filter"), err)
	}
	if got := filter["inboxes.id"]["$in"]; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("filter should restrict inboxes.id to [1 2], got %q", query.Get("filter"))
	}
	if !strings.Contains(text, `"Gold"`) {
		t.Errorf("result should carry the sideloaded option names, got %s", text)
	}
}

// TestCustomFieldListWithoutInboxesSendsNoFilter keeps an unfiltered call
// unfiltered.
func TestCustomFieldListWithoutInboxesSendsNoFilter(t *testing.T) {
	mcpServer, lastRequestURL, cleanup := testutil.DeskMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"customfields":[]}`))
	defer cleanup()

	lookupResultText(t, mcpServer, twdesk.MethodCustomFieldList, map[string]any{
		"inboxIDs": nil, "page": nil, "pageSize": nil, "orderBy": nil, "orderDirection": nil, "fields": nil,
	})

	requestURL := lastRequestURL()
	if query := requestURL.Query(); query.Has("filter") {
		t.Errorf("query parameter \"filter\" should be absent, got %q", query.Get("filter"))
	}
}

// lookupResultText runs a lookup tool and returns its text content, failing
// the test on an error result.
func lookupResultText(t *testing.T, mcpServer *mcp.Server, method toolsets.Method, args map[string]any) string {
	t.Helper()

	var text string
	testutil.ExecuteToolRequest(t, mcpServer, method.String(), args,
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()

			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("unexpected result type: %T", result)
			}
			if toolResult.IsError || len(toolResult.Content) == 0 {
				t.Fatalf("expected a successful result with content, got %+v", toolResult)
			}
			textContent, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("unexpected content type: %T", toolResult.Content[0])
			}
			text = textContent.Text
		}))
	return text
}
