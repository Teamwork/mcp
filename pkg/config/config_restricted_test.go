package config

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
)

const restrictedTestBody = `{"people":[{"id":777,"firstName":"John","email":"john@example.com",` +
	`"phone":"+1 555 0100","avatarUrl":"https://example.com/john.jpg","userCost":100,` +
	`"meta":{"notifiedUserIds":[12345]}}]}`

func restrictedTestHandler(_ context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
	var structured any
	_ = json.Unmarshal([]byte(restrictedTestBody), &structured)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: restrictedTestBody},
			&mcp.TextContent{Text: "not JSON: john@example.com"},
		},
		StructuredContent: structured,
	}, nil
}

// TestRestrictedDataMiddleware pins that a restricted installation's tool
// results lose every sensitive attribute, on both the text and the structured
// content, while any other installation's pass through untouched.
func TestRestrictedDataMiddleware(t *testing.T) {
	handler := restrictedDataMiddleware(restrictedTestHandler)
	sensitive := []string{"email", "phone", "avatarUrl", "userCost", "notifiedUserIds"}

	t.Run("restricted", func(t *testing.T) {
		ctx := twctx.WithRestrictedData(context.Background(), true)
		result, err := handler(ctx, "tools/call", nil)
		if err != nil {
			t.Fatal(err)
		}
		toolResult := result.(*mcp.CallToolResult)
		text := toolResult.Content[0].(*mcp.TextContent).Text
		structured, _ := json.Marshal(toolResult.StructuredContent)
		for _, got := range []string{text, string(structured)} {
			for _, key := range sensitive {
				if strings.Contains(got, key) {
					t.Errorf("%s reached the caller: %s", key, got)
				}
			}
			if !strings.Contains(got, `"firstName":"John"`) {
				t.Errorf("a non-sensitive attribute was dropped: %s", got)
			}
		}
		if prose := toolResult.Content[1].(*mcp.TextContent).Text; prose != "not JSON: john@example.com" {
			t.Errorf("a non-JSON block was rewritten: %s", prose)
		}
	})

	t.Run("not restricted", func(t *testing.T) {
		result, err := handler(context.Background(), "tools/call", nil)
		if err != nil {
			t.Fatal(err)
		}
		if text := result.(*mcp.CallToolResult).Content[0].(*mcp.TextContent).Text; text != restrictedTestBody {
			t.Errorf("an unrestricted result was changed: %s", text)
		}
	})
}

// TestNewMCPServerRestrictsToolResults pins that NewMCPServer wires the
// restricted-data middleware in: a tool call over a real session loses its
// sensitive attributes for a restricted installation only.
func TestNewMCPServerRestrictsToolResults(t *testing.T) {
	toolsets.RegisterToolOrder(nil)
	toolset := toolsets.NewToolset("restricted-read", "toolset used by the config tests")
	toolset.AddReadTools(toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        "restricted-read",
			Description: "tool used by the config tests",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		Handler: func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: restrictedTestBody}}}, nil
		},
	})
	group := toolsets.NewToolsetGroup(false)
	group.AddToolset(toolset)
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatal(err)
	}
	var resources Resources
	resources.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, restricted := range []bool{true, false} {
		ctx := twctx.WithRestrictedData(t.Context(), restricted)
		serverTransport, clientTransport := mcp.NewInMemoryTransports()
		serverSession, err := NewMCPServer(resources, group).Connect(ctx, serverTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
		clientSession, err := client.Connect(ctx, clientTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "restricted-read"})
		_ = clientSession.Close()
		_ = serverSession.Close()
		if err != nil {
			t.Fatal(err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if got := strings.Contains(text, "john@example.com"); got == restricted {
			t.Errorf("restricted=%v: email present=%v in %s", restricted, got, text)
		}
	}
}
