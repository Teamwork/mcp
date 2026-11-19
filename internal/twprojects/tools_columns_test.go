package twprojects_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/helpers"
	pkgtestutil "github.com/teamwork/mcp/pkg/testutil"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// columnsToolArgs gives the tools that resolve something with a second request
// the ID that skips it, since the mock answers every request with one body.
var columnsToolArgs = map[string]map[string]any{
	twprojects.MethodCalendarEventList.String(): {"calendar_id": 7},
}

// TestListToolsColumnsFormat walks every list tool. For each one it builds a
// response body from the tool's own output schema, so every list and sideload
// the schema names is filled, and checks the opt-in end to end:
//
//   - without it, NewMCPServer returns what the tool returns, byte for byte;
//   - with it, tools/list publishes a columns schema, and the text and
//     structured content of the result validate against it;
//   - the columns result holds the same records: expanded back into objects,
//     it equals the default result once its null attributes are removed.
func TestListToolsColumnsFormat(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var checked int
	for _, tool := range twprojectsListTools(t) {
		t.Run(tool.Tool.Name, func(t *testing.T) {
			spec := helpers.NewColumnSpec(tool.Tool.OutputSchema)
			if len(spec.Lists) == 0 {
				t.Fatal("the output schema names no list of records")
			}
			body, err := json.Marshal(sampleFromSchema(tool.Tool.OutputSchema.(*jsonschema.Schema), "", 0))
			if err != nil {
				t.Fatal(err)
			}
			args := toolArgsFromSchema(tool.Tool.InputSchema.(*jsonschema.Schema))
			maps.Copy(args, columnsToolArgs[tool.Tool.Name])

			plain := callListTool(t, tool.Tool.Name, args, body, "")
			objects := callListTool(t, tool.Tool.Name, args, body, helpers.ListFormatObjects)
			if objects.text != plain.text {
				t.Fatalf("the default result changed:\n got: %s\nwant: %s", objects.text, plain.text)
			}

			columns := callListTool(t, tool.Tool.Name, args, body, helpers.ListFormatColumns)
			checkColumnsContract(t, columns, spec)
			checkSameRecords(t, objects.text, columns.text, spec)
			checked++
		})
	}
	if checked == 0 {
		t.Fatal("no list tool was checked")
	}
}

// twprojectsListTools returns every list tool, with all toolsets enabled.
func twprojectsListTools(t *testing.T) []toolsets.ToolWrapper {
	t.Helper()
	group := twprojects.DefaultToolsetGroup(false, true, nil)
	var tools []toolsets.ToolWrapper
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			if strings.Contains(tool.Tool.Name, "-list_") {
				tools = append(tools, tool)
			}
		}
	}
	slices.SortFunc(tools, func(a, b toolsets.ToolWrapper) int { return strings.Compare(a.Tool.Name, b.Tool.Name) })
	if len(tools) < 30 {
		t.Fatalf("found %d list tools, which is fewer than expected", len(tools))
	}
	return tools
}

type listToolCall struct {
	text       string
	structured any
	schema     any
}

// callListTool calls a tool through config.NewMCPServer, with the list format
// as the server setting. An empty format calls a server without
// NewMCPServer's middleware, which is what the tool itself returns.
func callListTool(t *testing.T, name string, args map[string]any, body []byte, format string) listToolCall {
	t.Helper()
	group := twprojects.DefaultToolsetGroup(false, true, pkgtestutil.EngineMock(http.StatusOK, body))
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatal(err)
	}
	server := pkgtestutil.MCPServer(t, group)
	if format != "" {
		var resources config.Resources
		resources.Info.ListFormat = format
		server = config.NewMCPServer(resources, group)
	}
	session := connectTestSession(t, server)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	pkgtestutil.CheckMessage(t, result)
	call := listToolCall{
		text:       result.Content[0].(*mcp.TextContent).Text,
		structured: result.StructuredContent,
	}
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == name {
			call.schema = tool.OutputSchema
		}
	}
	return call
}

func connectTestSession(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// checkColumnsContract validates the text and the structured content of a
// columns result against the output schema tools/list published with it.
func checkColumnsContract(t *testing.T, call listToolCall, spec helpers.ColumnSpec) {
	t.Helper()
	resolved := resolvePublishedSchema(t, call.schema)
	var fromText map[string]any
	if err := json.Unmarshal([]byte(call.text), &fromText); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"text": fromText, "structured": call.structured} {
		if err := resolved.Validate(value); err != nil {
			t.Errorf("the %s does not match the columns schema: %v", name, err)
		}
	}
	for _, key := range spec.Lists {
		if _, sent := fromText[key]; !sent {
			continue
		}
		if list, ok := fromText[key].(map[string]any); !ok || list["rows"] == nil {
			t.Errorf("%s is not in the columns format: %v", key, fromText[key])
		}
	}
}

// resolvePublishedSchema resolves an output schema as tools/list sent it, and
// fails when it is not the columns schema.
func resolvePublishedSchema(t *testing.T, published any) *jsonschema.Resolved {
	t.Helper()
	encoded, err := json.Marshal(published)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"rows"`) {
		t.Fatalf("tools/list did not publish the columns schema: %s", encoded)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// checkSameRecords expands each list in the columns result back into objects
// and compares it with the default result. A null cell and a missing
// attribute mean the same, so nulls are removed from both sides first.
func checkSameRecords(t *testing.T, objectsText, columnsText string, spec helpers.ColumnSpec) {
	t.Helper()
	var objects, columns map[string]any
	if err := json.Unmarshal([]byte(objectsText), &objects); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(columnsText), &columns); err != nil {
		t.Fatal(err)
	}
	for _, key := range spec.Lists {
		// A schema can name a list the tool does not send, such as the
		// suggestions of list_project_templates. It stays absent.
		if _, sent := objects[key]; !sent {
			if _, sent := columns[key]; sent {
				t.Errorf("%s was added: %v", key, columns[key])
			}
			continue
		}
		compareRecords(t, key, objects[key], expandColumns(t, columns[key]))
	}
	included, _ := objects["included"].(map[string]any)
	columnsIncluded, _ := columns["included"].(map[string]any)
	for name, section := range included {
		var want []any
		for _, id := range slices.Sorted(maps.Keys(section.(map[string]any))) {
			want = append(want, section.(map[string]any)[id])
		}
		compareRecords(t, "included."+name, want, expandColumns(t, columnsIncluded[name]))
	}
}

func compareRecords(t *testing.T, key string, want, got any) {
	t.Helper()
	want, got = withoutNulls(want), withoutNulls(got)
	if !reflect.DeepEqual(want, got) {
		wantJSON, _ := json.Marshal(want)
		gotJSON, _ := json.Marshal(got)
		t.Errorf("%s holds other records:\n got: %s\nwant: %s", key, gotJSON, wantJSON)
	}
}

// expandColumns turns a columns object back into a list of objects, and each
// flattened relationship back into {id, type}.
func expandColumns(t *testing.T, value any) []any {
	t.Helper()
	list, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("not a columns object: %v", value)
	}
	names, _ := list["columns"].([]any)
	relationships, _ := list["relationships"].(map[string]any)
	rows, ok := list["rows"].([]any)
	if !ok {
		t.Fatalf("no rows: %v", value)
	}
	var records []any
	for _, row := range rows {
		record := map[string]any{}
		for i, cell := range row.([]any) {
			name := names[i].(string)
			record[name] = cell
			if typ, ok := relationships[name].(string); ok {
				record[name] = expandRelationship(cell, typ)
			}
		}
		records = append(records, record)
	}
	return records
}

func expandRelationship(cell any, typ string) any {
	switch typed := cell.(type) {
	case nil:
		return nil
	case []any:
		expanded := make([]any, len(typed))
		for i, id := range typed {
			expanded[i] = map[string]any{"id": id, "type": typ}
		}
		return expanded
	}
	return map[string]any{"id": cell, "type": typ}
}

// withoutNulls removes the null attributes of every object, at any depth.
func withoutNulls(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		object := map[string]any{}
		for key, element := range typed {
			if element != nil {
				object[key] = withoutNulls(element)
			}
		}
		return object
	case []any:
		list := make([]any, len(typed))
		for i, element := range typed {
			list[i] = withoutNulls(element)
		}
		return list
	}
	return value
}

// sampleFromSchema builds a value that fills every property of a schema. A
// list holds two records. Each section under "included" is a map of ID to
// record, which is what the endpoints send, whatever the schema says.
func sampleFromSchema(schema *jsonschema.Schema, name string, depth int) any {
	if schema == nil || depth > 6 {
		return nil
	}
	switch sampleType(schema) {
	case "object":
		return sampleObject(schema, name, depth)
	case "array":
		return []any{sampleFromSchema(schema.Items, name, depth+1), sampleFromSchema(schema.Items, name, depth+1)}
	case "integer", "number":
		return 7
	case "boolean":
		// false, so that a tool reading hasMore does not page for ever.
		return false
	case "string":
		return sampleString(schema, name)
	}
	return nil
}

func sampleObject(schema *jsonschema.Schema, name string, depth int) any {
	if depth == 1 && name == "included" {
		included := map[string]any{"users": sampleSideload()}
		for section := range schema.Properties {
			included[section] = sampleSideload()
		}
		return included
	}
	object := map[string]any{}
	for property, propertySchema := range schema.Properties {
		// count is the count_only answer, never sent beside the rows.
		if depth == 0 && property == "count" {
			continue
		}
		object[property] = sampleFromSchema(propertySchema, property, depth+1)
	}
	return object
}

func sampleSideload() map[string]any {
	return map[string]any{
		"7": map[string]any{"id": 7, "name": "Seven"},
		"9": map[string]any{"id": 9, "name": "Nine", "project": map[string]any{"id": 7, "type": "projects"}},
	}
}

func sampleType(schema *jsonschema.Schema) string {
	if schema.Type != "" {
		return schema.Type
	}
	for _, typ := range schema.Types {
		if typ != "null" {
			return typ
		}
	}
	if len(schema.AnyOf) > 0 {
		return sampleType(schema.AnyOf[0])
	}
	return ""
}

func sampleString(schema *jsonschema.Schema, name string) any {
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}
	switch {
	case schema.Format == "date":
		return "2026-11-19"
	case schema.Format == "date-time", strings.HasSuffix(name, "At"), strings.HasSuffix(name, "Date"), strings.HasSuffix(name, "On"):
		return "2026-11-19T09:00:00Z"
	}
	return "text"
}

// toolArgsFromSchema fills the required parameters of a tool.
func toolArgsFromSchema(schema *jsonschema.Schema) map[string]any {
	args := map[string]any{}
	for _, name := range schema.Required {
		property := schema.Properties[name]
		switch sampleType(property) {
		case "integer", "number":
			args[name] = 7
		case "array":
			args[name] = []any{7}
		case "boolean":
			args[name] = true
		default:
			args[name] = fmt.Sprint(sampleString(property, name))
		}
	}
	return args
}

// TestListToolsColumnsFormatIgnoresCountOnly pins that a count_only answer,
// which has no rows, comes back as it is.
func TestListToolsColumnsFormatIgnoresCountOnly(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	body := []byte(`{"tasks":[],"meta":{"page":{"count":42}}}`)
	columns := callListTool(t, twprojects.MethodTaskList.String(), map[string]any{"count_only": true}, body,
		helpers.ListFormatColumns)
	if columns.text != `{"count":42}` {
		t.Errorf("count_only result = %s", columns.text)
	}
}
