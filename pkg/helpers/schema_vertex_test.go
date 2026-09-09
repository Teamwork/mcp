package helpers_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
)

// TestVertexSchema covers the input-side strip: Vertex has no `examples` field
// and an undocumented `format` buys nothing there.
func TestVertexSchema(t *testing.T) {
	safe := helpers.VertexSchema(&jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"created_after": {
				Type:     "string",
				Format:   "date",
				Examples: []any{"2026-08-03"},
			},
			"updated_after": {Type: "string", Format: "date-time"},
			"rows": {Type: "array", Items: &jsonschema.Schema{
				Type:     "object",
				Examples: []any{"nested"},
			}},
			"either": {AnyOf: []*jsonschema.Schema{
				{Type: "string", Format: "date", Examples: []any{"x"}},
			}},
		},
	})

	if got := safe.Properties["created_after"].Examples; got != nil {
		t.Errorf("examples = %v, want nil", got)
	}
	if got := safe.Properties["created_after"].Format; got != "" {
		t.Errorf("format = %q, want it stripped", got)
	}
	if got := safe.Properties["updated_after"].Format; got != "date-time" {
		t.Errorf("format = %q, want date-time kept", got)
	}
	if got := safe.Properties["rows"].Items.Examples; got != nil {
		t.Errorf("nested examples = %v, want nil", got)
	}
	if got := safe.Properties["either"].AnyOf[0]; got.Examples != nil || got.Format != "" {
		t.Errorf("branch = %+v, want examples and format stripped", got)
	}
	// The strip must not touch anything else.
	if safe.Properties["created_after"].Type != "string" {
		t.Errorf("type = %q, want string", safe.Properties["created_after"].Type)
	}
}

// TestVertexSchemaDoesNotTouchTheOriginal mirrors the strict-variant guard:
// tools/list is handed the registered pointers.
func TestVertexSchemaDoesNotTouchTheOriginal(t *testing.T) {
	published := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"created_after": {Type: "string", Format: "date", Examples: []any{"2026-08-03"}},
		},
	}
	before, err := json.Marshal(published)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	_ = helpers.VertexSchema(published)

	after, err := json.Marshal(published)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("the published schema was modified:\n before %s\n after  %s", before, after)
	}
}

// TestVertexToolsDropsTheOutputSchema is the fix for the remaining blocker: an
// output schema declares type unions Vertex cannot parse, and their nulls are
// real wire values so they cannot be removed. Dropping the schema loads the
// tools and only gives up validation of structuredContent.
func TestVertexToolsDropsTheOutputSchema(t *testing.T) {
	input := &jsonschema.Schema{Type: "object"}
	output := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"deletedAt": {Types: []string{"string", "null"}},
		},
	}
	tool := &mcp.Tool{Name: "twprojects-get_team", InputSchema: input, OutputSchema: output}

	safe := helpers.VertexTools([]*mcp.Tool{tool})
	if len(safe) != 1 {
		t.Fatalf("got %d tools, want 1", len(safe))
	}
	if safe[0].OutputSchema != nil {
		t.Errorf("output schema = %v, want it dropped", safe[0].OutputSchema)
	}
	if safe[0] == tool {
		t.Error("the tool itself was not copied")
	}
	if tool.OutputSchema != any(output) {
		t.Error("the registered tool lost its output schema")
	}
	if safe[0].Name != tool.Name {
		t.Errorf("name = %q, want it carried over", safe[0].Name)
	}
}

// TestVertexToolsPassesThroughForeignSchemas keeps a tool whose input schema is
// not a *jsonschema.Schema, still dropping its output schema.
func TestVertexToolsPassesThroughForeignSchemas(t *testing.T) {
	tool := &mcp.Tool{
		Name:         "twprojects-list_tasks",
		InputSchema:  map[string]any{"type": "object"},
		OutputSchema: &jsonschema.Schema{Type: "object"},
	}
	safe := helpers.VertexTools([]*mcp.Tool{tool})
	if len(safe) != 1 {
		t.Fatalf("got %d tools, want 1", len(safe))
	}
	if safe[0].OutputSchema != nil {
		t.Error("output schema was not dropped")
	}
	if _, ok := safe[0].InputSchema.(map[string]any); !ok {
		t.Errorf("input schema = %T, want it passed through", safe[0].InputSchema)
	}
}

// TestVertexToolsDoesNotTruncate pins the decision not to cap the tool count.
// The 128 ceiling is on what a client sends the model, and Gemini Enterprise
// enables actions individually, so cutting the list here would hide tools for
// no gain — and unevenly, since the preference order names only 20 tools and a
// cut at 128 removes every twspaces tool.
func TestVertexToolsDoesNotTruncate(t *testing.T) {
	tools := make([]*mcp.Tool, helpers.VertexMaxFunctionDeclarations+50)
	for i := range tools {
		tools[i] = &mcp.Tool{
			Name:        fmt.Sprintf("twprojects-tool_%03d", i),
			InputSchema: &jsonschema.Schema{Type: "object"},
		}
	}

	safe := helpers.VertexTools(tools)
	if len(safe) != len(tools) {
		t.Errorf("got %d tools, want all %d", len(safe), len(tools))
	}
	for i, tool := range safe {
		if want := fmt.Sprintf("twprojects-tool_%03d", i); tool.Name != want {
			t.Fatalf("tool %d = %q, want %q", i, tool.Name, want)
		}
	}
}

// TestVertexToolsKeepsAShortList is the ordinary case.
func TestVertexToolsKeepsAShortList(t *testing.T) {
	tools := []*mcp.Tool{
		{Name: "twdesk-get_ticket", InputSchema: &jsonschema.Schema{Type: "object"}},
		{Name: "twdesk-list_inboxes", InputSchema: &jsonschema.Schema{Type: "object"}},
	}
	if safe := helpers.VertexTools(tools); len(safe) != 2 {
		t.Errorf("got %d tools, want 2", len(safe))
	}
}
