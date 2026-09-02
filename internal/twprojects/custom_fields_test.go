package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomFieldCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldCreate.String(), map[string]any{
		"name":        "Priority Score",
		"type":        "number-integer",
		"entity":      "task",
		"description": "Priority score for tasks",
		"required":    true,
		"project_id":  float64(456),
	})
}

func TestCustomFieldCreateDropdown(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldCreate.String(), map[string]any{
		"name":   "Status",
		"type":   "dropdown",
		"entity": "task",
		"options": map[string]any{
			"choices": []any{
				map[string]any{"value": "Open", "color": "#ff0000"},
				map[string]any{"value": "Closed", "color": "#00ff00"},
			},
		},
	})
}

func TestCustomFieldUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldUpdate.String(), map[string]any{
		"id":          float64(123),
		"name":        "Updated name",
		"description": "Updated description",
	})
}

func TestCustomFieldDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldDelete.String(), map[string]any{
		"id": float64(123),
	})
}

// TestCustomFieldChoiceColorIsOptional covers a dropdown choice stored without
// a colour, which the web UI allows. The endpoint models the choice colour with
// its own non-optional type, so it answers such a choice with the bare "#", and
// the read used to fail the whole call: the choice colour was a
// twapi.HexColor, which rejects that, and the error propagates out of
// CustomField's own UnmarshalJSON. It is a twapi.OptionalHexColor now, so the
// colour reads as unset and the key is left out of the response.
//
// The colour also decides which spelling a caller sees, the same way it does on
// allocations: the get re-encodes the typed value and so restores the "#",
// while the list streams the body and passes the endpoint's own spelling
// through.
func TestCustomFieldChoiceColorIsOptional(t *testing.T) {
	const field = `{"id":123,"type":"dropdown","options":{"choices":[` +
		`{"value":"Blocked","color":"#"},{"value":"Shipped","color":"#8bc34a"}]}}`

	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfield":`+field+`}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldGet.String(), map[string]any{
		"id": float64(123),
	}, testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
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

		var decoded struct {
			CustomField struct {
				Options struct {
					Choices []map[string]any `json:"choices"`
				} `json:"options"`
			} `json:"customfield"`
		}
		if err := json.Unmarshal([]byte(content.Text), &decoded); err != nil {
			t.Fatalf("failed to decode the response: %s", err)
		}

		choices := decoded.CustomField.Options.Choices
		if len(choices) != 2 {
			t.Fatalf("expected 2 choices but got %d: %s", len(choices), content.Text)
		}
		if color, ok := choices[0]["color"]; ok {
			t.Errorf("expected the colourless choice to carry no color but got %v", color)
		}
		if got := choices[1]["color"]; got != "#8bc34a" {
			t.Errorf("expected color #8bc34a but got %v", got)
		}
	}))
}

func TestCustomFieldGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfield":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCustomFieldList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfields":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldList.String(), map[string]any{
		"search_term":     "priority",
		"ids":             []int64{1, 2, 3},
		"entities":        []string{"task", "project"},
		"project_ids":     []int64{10},
		"only_site_level": false,
		"order_by":        "name",
		"order_mode":      "asc",
		"page":            float64(1),
		"page_size":       float64(10),
	})
}
