package helpers_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/pkg/helpers"
)

func TestToColumns(t *testing.T) {
	spec := helpers.ColumnSpec{Lists: []string{"tasks"}, Included: true}
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "sends each attribute name once and fills a missing one with null",
			in:   `{"tasks":[{"name":"a","id":1},{"id":2,"dueDate":"2026-11-19"}]}`,
			want: `{"tasks":{"columns":["id","dueDate","name"],"rows":[[1,null,"a"],[2,"2026-11-19",null]]}}`,
		},
		{
			name: "flattens relationships of one type to IDs",
			in: `{"tasks":[` +
				`{"id":1,"tasklist":{"id":7,"type":"tasklists"},"assignees":[{"id":5,"type":"users"}]},` +
				`{"id":2,"tasklist":null,"assignees":[]}]}`,
			want: `{"tasks":{"columns":["id","assignees","tasklist"],` +
				`"relationships":{"assignees":"users","tasklist":"tasklists"},` +
				`"rows":[[1,[5],7],[2,[],null]]}}`,
		},
		{
			name: "keeps relationships of more than one type as objects",
			in: `{"tasks":[{"id":1,"assignees":[{"id":5,"type":"users"}]},` +
				`{"id":2,"assignees":[{"id":3,"type":"teams"}]}]}`,
			want: `{"tasks":{"columns":["id","assignees"],` +
				`"rows":[[1,[{"id":5,"type":"users"}]],[2,[{"id":3,"type":"teams"}]]]}}`,
		},
		{
			name: "keeps an object that is more than a relationship",
			in:   `{"tasks":[{"id":1,"card":{"id":4,"type":"cards","column":2}}]}`,
			want: `{"tasks":{"columns":["id","card"],"rows":[[1,{"column":2,"id":4,"type":"cards"}]]}}`,
		},
		{
			name: "changes each sideload and orders it by numeric ID",
			in: `{"tasks":[],"included":{"users":{"10":{"id":10,"firstName":"Jane"},` +
				`"9":{"id":9,"firstName":"John"}}}}`,
			want: `{"included":{"users":{"columns":["id","firstName"],"rows":[[9,"John"],[10,"Jane"]]}},` +
				`"tasks":{"columns":[],"rows":[]}}`,
		},
		{
			name: "leaves meta, unnamed keys and a null list as they are",
			in:   `{"tasks":null,"meta":{"page":{"count":2}},"other":[{"id":1}]}`,
			want: `{"meta":{"page":{"count":2}},"other":[{"id":1}],"tasks":null}`,
		},
		{
			name: "copies numbers as written",
			in:   `{"tasks":[{"id":9007199254740993,"progress":1.50}]}`,
			want: `{"tasks":{"columns":["id","progress"],"rows":[[9007199254740993,1.50]]}}`,
		},
		{
			name: "leaves a list that holds something other than records",
			in:   `{"tasks":[1,2]}`,
			want: `{"tasks":[1,2]}`,
		},
		{
			name: "returns data that is not a JSON object unchanged",
			in:   `not JSON`,
			want: `not JSON`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := helpers.ToColumns([]byte(tt.in), spec)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("ToColumns()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

// columnsTestSchema is a list tool output schema in the shape the twprojects
// list tools publish.
func columnsTestSchema() *jsonschema.Schema {
	task := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"id":   {Type: "integer"},
		"name": {Type: "string"},
	}}
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"tasks":    {Types: []string{"null", "array"}, Items: task},
			"tags":     {Type: "array", Items: &jsonschema.Schema{Type: "string"}},
			"meta":     {Type: "object"},
			"count":    {Type: "integer"},
			"included": {Type: "object", Properties: map[string]*jsonschema.Schema{"projects": {Type: "object"}}},
		},
	}
}

func TestNewColumnSpec(t *testing.T) {
	got := helpers.NewColumnSpec(columnsTestSchema())
	want := helpers.ColumnSpec{Lists: []string{"tasks"}, Included: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewColumnSpec() = %+v, want %+v", got, want)
	}
	if spec := helpers.NewColumnSpec(nil); !spec.IsZero() {
		t.Errorf("a tool with no output schema got %+v", spec)
	}
}

// TestColumnsSchemaDoesNotTouchTheOriginal pins that the columns schema is a
// copy: tools/list hands out the registered schemas, so a change in place
// would reach every client.
func TestColumnsSchemaDoesNotTouchTheOriginal(t *testing.T) {
	original := columnsTestSchema()
	before, _ := json.Marshal(original)
	columns := helpers.ColumnsSchema(original, helpers.NewColumnSpec(original))
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatalf("the original schema changed:\nbefore: %s\n after: %s", before, after)
	}
	if columns.Properties["tasks"].Properties["rows"] == nil {
		t.Errorf("tasks is not described as columns: %+v", columns.Properties["tasks"])
	}
	if columns.Properties["tags"].Type != "array" {
		t.Errorf("a list of strings was changed: %+v", columns.Properties["tags"])
	}
}

// TestColumnsToolResultMatchesColumnsSchema pins the contract: a result in the
// columns format validates against the schema published for it, in its text
// and in its structured content.
func TestColumnsToolResultMatchesColumnsSchema(t *testing.T) {
	schema := columnsTestSchema()
	spec := helpers.NewColumnSpec(schema)
	body := `{"tasks":[{"id":1,"name":"a"},{"id":2}],"tags":["x"],"meta":{},` +
		`"included":{"projects":{"7":{"id":7,"name":"Website"}},"users":{"5":{"id":5}}}}`
	var structured any
	_ = json.Unmarshal([]byte(body), &structured)
	result := &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: body}},
		StructuredContent: structured,
	}
	if err := helpers.ColumnsToolResult(result, spec); err != nil {
		t.Fatal(err)
	}

	resolved, err := helpers.ColumnsSchema(schema, spec).Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var fromText any
	if err := json.Unmarshal([]byte(text), &fromText); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"text": fromText, "structured": result.StructuredContent} {
		if err := resolved.Validate(value); err != nil {
			t.Errorf("%s does not match the columns schema: %v", name, err)
		}
	}
	if !strings.Contains(text, `"users":{"columns":["id"],"rows":[[5]]}`) {
		t.Errorf("a sideload the schema does not name was not changed: %s", text)
	}
}

func TestColumnsToolResultLeavesOtherResults(t *testing.T) {
	spec := helpers.ColumnSpec{Lists: []string{"tasks"}}
	body := `{"tasks":[{"id":1}]}`
	results := map[string]*mcp.CallToolResult{
		"error": {IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: body}}},
		"two blocks": {Content: []mcp.Content{
			&mcp.TextContent{Text: body}, &mcp.TextContent{Text: body},
		}},
	}
	for name, result := range results {
		if err := helpers.ColumnsToolResult(result, spec); err != nil {
			t.Fatal(err)
		}
		if text := result.Content[0].(*mcp.TextContent).Text; text != body {
			t.Errorf("%s: result changed to %s", name, text)
		}
	}
}

func TestColumnsToolsCopiesOnlyNamedTools(t *testing.T) {
	schema := columnsTestSchema()
	listTool := &mcp.Tool{Name: "twprojects-list_tasks", OutputSchema: schema}
	getTool := &mcp.Tool{Name: "twprojects-get_task", OutputSchema: schema}
	specs := map[string]helpers.ColumnSpec{listTool.Name: helpers.NewColumnSpec(schema)}

	got := helpers.ColumnsTools([]*mcp.Tool{listTool, getTool}, specs)
	if got[0] == listTool || got[0].OutputSchema == schema {
		t.Error("the list tool was not copied")
	}
	if got[1] != getTool {
		t.Error("a tool with no spec was copied")
	}
	if listTool.OutputSchema != schema {
		t.Error("the registered list tool changed")
	}
}
