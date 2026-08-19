package config

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
)

// TestListToolsCacheScopeIsPrivate guards a cross-tenant leak. The tool list is
// filtered per OAuth token scope, but the SDK marks cacheable results "public"
// by default (SEP-2549), which would let a shared intermediary serve one
// tenant's tool list to another.
func TestListToolsCacheScopeIsPrivate(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		// wantTools is the set of tool names the caller should be able to see.
		wantTools []string
	}{{
		name:      "no scopes, unfiltered list",
		scopes:    nil,
		wantTools: []string{"twdesk-read", "twprojects-read"},
	}, {
		name:      "projects scope only",
		scopes:    []string{"projects"},
		wantTools: []string{"twprojects-read"},
	}, {
		name:      "scope matching no tool",
		scopes:    []string{"spaces"},
		wantTools: nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.scopes != nil {
				ctx = twctx.WithScopes(ctx, tt.scopes)
			}

			result := listTools(ctx, t)

			if got := result.CacheScope; got != cacheScopePrivate {
				t.Errorf("tools/list CacheScope = %q, want %q", got, cacheScopePrivate)
			}

			got := make([]string, 0, len(result.Tools))
			for _, tool := range result.Tools {
				got = append(got, tool.Name)
			}
			if len(got) != len(tt.wantTools) {
				t.Fatalf("tools = %v, want %v", got, tt.wantTools)
			}
			for i, want := range tt.wantTools {
				if got[i] != want {
					t.Errorf("tools = %v, want %v", got, tt.wantTools)
					break
				}
			}
		})
	}
}

// TestCapabilitiesOmitLogging pins the decision to stop advertising the
// deprecated "logging" capability (SEP-2577), which this server never honoured.
// Old clients are unaffected: on pre-2026-07-28 revisions the SDK still answers
// "logging/setLevel". On 2026-07-28 it rejects the method, but only because the
// revision removed it, not because of this change.
func TestCapabilitiesOmitLogging(t *testing.T) {
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := newTestMCPServer(t).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	defer serverSession.Close() //nolint:errcheck

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer clientSession.Close() //nolint:errcheck

	capabilities := clientSession.InitializeResult().Capabilities
	if capabilities.Logging != nil { //nolint:staticcheck // asserting the deprecated capability is absent
		t.Error("server advertises the deprecated logging capability, want it omitted")
	}
	if capabilities.Tools == nil {
		t.Error("server does not advertise the tools capability, want it advertised")
	}
	if _, ok := capabilities.Extensions["io.modelcontextprotocol/ui"]; !ok {
		t.Error("server does not advertise the MCP Apps UI extension, want it advertised")
	}
}

// TestCapabilitiesOmitListChanged pins the list-changed capabilities off. This
// server never pushes list-change notifications, and advertising them invites
// clients to open a long-lived "subscriptions/listen" stream (SEP-2575) that a
// stateless, load-balanced deployment would hold open forever with nothing to
// send. The SDK turns these on by default whenever the matching capability
// struct is left nil, so this test guards against that default leaking in.
func TestCapabilitiesOmitListChanged(t *testing.T) {
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := newTestMCPServer(t).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	defer serverSession.Close() //nolint:errcheck

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer clientSession.Close() //nolint:errcheck

	capabilities := clientSession.InitializeResult().Capabilities
	if capabilities.Tools == nil || capabilities.Tools.ListChanged {
		t.Error("tools.listChanged is advertised, want it off")
	}
	if capabilities.Prompts == nil || capabilities.Prompts.ListChanged {
		t.Error("prompts.listChanged is advertised, want it off")
	}
	if capabilities.Resources == nil || capabilities.Resources.ListChanged {
		t.Error("resources.listChanged is advertised, want it off")
	}
	if capabilities.Resources != nil && capabilities.Resources.Subscribe {
		t.Error("resources.subscribe is advertised, want it off")
	}
}

// listTools runs a tools/list round trip against a server built by
// NewMCPServer, so the receiving middleware under test is exercised end to end.
func listTools(ctx context.Context, t *testing.T) *mcp.ListToolsResult {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := newTestMCPServer(t).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	defer serverSession.Close() //nolint:errcheck

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer clientSession.Close() //nolint:errcheck

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	return result
}

// newTestMCPServer builds a server through NewMCPServer with one read tool per
// product prefix, so scope filtering has something to filter, plus a prompt and
// a resource so the prompt and resource capabilities are populated the way the
// real toolset groups populate them.
func newTestMCPServer(t *testing.T) *mcp.Server {
	t.Helper()

	toolsets.RegisterToolOrder(nil)

	projectsToolset := toolsets.NewToolset("test-projects", "toolset used by the config tests")
	projectsToolset.AddReadTools(newTestReadTool("twprojects-read"))
	projectsToolset.AddPrompts(toolsets.NewServerPrompt(
		&mcp.Prompt{Name: "twprojects-test_prompt"},
		func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{}, nil
		},
	))
	projectsToolset.AddResources(toolsets.NewServerResource(
		&mcp.Resource{Name: "twprojects-test_resource", URI: "ui://teamwork/test"},
		func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{}, nil
		},
	))

	deskToolset := toolsets.NewToolset("test-desk", "toolset used by the config tests")
	deskToolset.AddReadTools(newTestReadTool("twdesk-read"))

	// One group per namespace, as the real server builds them: the tools/list
	// scope filter reads the scope off the group, so a group is the unit a
	// scope applies to.
	projectsGroup := toolsets.NewToolsetGroup(false).SetNamespace("twprojects", "projects")
	projectsGroup.AddToolset(projectsToolset)
	deskGroup := toolsets.NewToolsetGroup(false).SetNamespace("twdesk", "desk")
	deskGroup.AddToolset(deskToolset)

	for _, group := range []*toolsets.ToolsetGroup{projectsGroup, deskGroup} {
		if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
			t.Fatalf("failed to enable toolsets: %v", err)
		}
	}

	var resources Resources
	resources.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewMCPServer(resources, projectsGroup, deskGroup)
}

func newTestReadTool(name string) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        name,
			Description: "tool used by the config tests",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		Handler: func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{}, nil
		},
	}
}
