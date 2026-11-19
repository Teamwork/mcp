package config

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/request"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
)

const columnsTestBody = `{"people":[{"id":777,"firstName":"John","email":"john@example.com",` +
	`"company":{"id":12345,"type":"companies"}}],"meta":{"page":{"count":1}}}`

func TestWantsColumnLists(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		header  string
		want    bool
	}{
		{name: "default"},
		{name: "server setting", setting: "columns", want: true},
		{name: "header", setting: "objects", header: "columns", want: true},
		{name: "header in another case", header: "Columns", want: true},
		{name: "header turns the server setting off", setting: "columns", header: "objects"},
		{name: "unknown value gets the default", header: "csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resources Resources
			resources.Info.ListFormat = tt.setting
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.header != "" {
				r.Header.Set(listFormatHeader, tt.header)
			}
			ctx := request.WithInfo(r.Context(), request.NewInfo(r))
			if got := wantsColumnLists(ctx, resources); got != tt.want {
				t.Errorf("wantsColumnLists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsListTool(t *testing.T) {
	for name, want := range map[string]bool{
		"twprojects-list_tasks":      true,
		"twdesk-list_users":          true,
		"twprojects-get_task":        false,
		"twprojects-count_tasks":     false,
		"twprojects-users_workload":  false,
		"list_tasks":                 false,
		"twprojects-search":          false,
		"twprojects-list_milestones": true,
	} {
		if got := isListTool(name); got != want {
			t.Errorf("isListTool(%q) = %v, want %v", name, got, want)
		}
	}
}

// columnsTestGroup has one list tool, one get tool with the same body and
// schema, and one list tool with no output schema.
func columnsTestGroup(t *testing.T) *toolsets.ToolsetGroup {
	t.Helper()
	toolsets.RegisterToolOrder(nil)
	schema := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"people": {Type: "array", Items: &jsonschema.Schema{Type: "object"}},
		"meta":   {Type: "object"},
	}}
	handler := func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var structured any
		_ = json.Unmarshal([]byte(columnsTestBody), &structured)
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: columnsTestBody}},
			StructuredContent: structured,
		}, nil
	}
	tool := func(name string, output any) toolsets.ToolWrapper {
		return toolsets.ToolWrapper{
			Tool: &mcp.Tool{
				Name:         name,
				Description:  "tool used by the config tests",
				Annotations:  &mcp.ToolAnnotations{ReadOnlyHint: true},
				InputSchema:  &jsonschema.Schema{Type: "object"},
				OutputSchema: output,
			},
			Handler: handler,
		}
	}
	toolset := toolsets.NewToolset("columns-read", "toolset used by the config tests")
	toolset.AddReadTools(
		tool("columns-list_people", schema),
		tool("columns-get_person", schema),
		tool("columns-list_untyped", nil),
	)
	group := toolsets.NewToolsetGroup(false)
	group.AddToolset(toolset)
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatal(err)
	}
	return group
}

// columnsTestSession connects a client to NewMCPServer with the given list
// format setting.
func columnsTestSession(t *testing.T, ctx context.Context, setting string) *mcp.ClientSession {
	t.Helper()
	var resources Resources
	resources.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	resources.Info.ListFormat = setting
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewMCPServer(resources, columnsTestGroup(t)).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func callColumnsTestTool(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string) string {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return result.Content[0].(*mcp.TextContent).Text
}

// TestNewMCPServerColumnLists pins the opt-in: without it every result is
// byte for byte what the tool returned, and with it only the list tool that
// publishes an output schema changes.
func TestNewMCPServerColumnLists(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		session := columnsTestSession(t, t.Context(), "")
		for _, name := range []string{"columns-list_people", "columns-get_person", "columns-list_untyped"} {
			if got := callColumnsTestTool(t, t.Context(), session, name); got != columnsTestBody {
				t.Errorf("%s changed without the opt-in: %s", name, got)
			}
		}
	})

	t.Run("columns", func(t *testing.T) {
		session := columnsTestSession(t, t.Context(), helpers.ListFormatColumns)
		want := `{"meta":{"page":{"count":1}},"people":{"columns":["id","company","email","firstName"],` +
			`"relationships":{"company":"companies"},"rows":[[777,12345,"john@example.com","John"]]}}`
		if got := callColumnsTestTool(t, t.Context(), session, "columns-list_people"); got != want {
			t.Errorf("list result:\n got: %s\nwant: %s", got, want)
		}
		for _, name := range []string{"columns-get_person", "columns-list_untyped"} {
			if got := callColumnsTestTool(t, t.Context(), session, name); got != columnsTestBody {
				t.Errorf("%s changed: %s", name, got)
			}
		}
	})
}

// TestNewMCPServerColumnListsSchema pins that tools/list publishes the columns
// output schema only to a request that asks for the format.
func TestNewMCPServerColumnListsSchema(t *testing.T) {
	for _, setting := range []string{"", helpers.ListFormatColumns} {
		session := columnsTestSession(t, t.Context(), setting)
		tools, err := session.ListTools(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, tool := range tools.Tools {
			encoded, _ := json.Marshal(tool.OutputSchema)
			hasColumns := strings.Contains(string(encoded), `"rows"`)
			want := setting == helpers.ListFormatColumns && tool.Name == "columns-list_people"
			if hasColumns != want {
				t.Errorf("setting %q: %s publishes the columns schema = %v: %s", setting, tool.Name, hasColumns, encoded)
			}
		}
	}
}

// TestNewMCPServerColumnListsAfterRestriction pins the middleware order. In
// the columns format a key becomes a column name, so the restricted keys must
// be removed before the change, or "email" would reach the caller.
func TestNewMCPServerColumnListsAfterRestriction(t *testing.T) {
	ctx := twctx.WithRestrictedData(t.Context(), true)
	session := columnsTestSession(t, ctx, helpers.ListFormatColumns)
	got := callColumnsTestTool(t, ctx, session, "columns-list_people")
	if strings.Contains(got, "email") || strings.Contains(got, "john@example.com") {
		t.Errorf("restricted data reached the caller: %s", got)
	}
	if !strings.Contains(got, `"columns":["id","company","firstName"]`) {
		t.Errorf("the result is not in the columns format: %s", got)
	}
}
