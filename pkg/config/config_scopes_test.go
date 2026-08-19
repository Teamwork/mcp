package config

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
)

// TestListToolsFiltersByGroupNamespace pins that the tools/list scope filter is
// driven by what each ToolsetGroup declares through SetNamespace, not by a
// hardcoded list of products. A server built on this package registers its own
// groups, and a scope the filter did not know about would list those tools to
// every token regardless of what it was granted.
func TestListToolsFiltersByGroupNamespace(t *testing.T) {
	tests := []struct {
		name      string
		scopes    []string
		wantTools []string
	}{{
		// A token carrying no scopes at all is not an authorisation decision this
		// filter makes; the list is left alone.
		name:      "no scopes leaves the list alone",
		scopes:    nil,
		wantTools: []string{"twpro-read", "unprefixed-read", "twprojects-read"},
	}, {
		name:      "a granted scope shows its group",
		scopes:    []string{"pro"},
		wantTools: []string{"twpro-read", "unprefixed-read"},
	}, {
		name:      "an ungranted scope hides its group",
		scopes:    []string{"projects"},
		wantTools: []string{"unprefixed-read", "twprojects-read"},
	}, {
		// The premium case: a group nobody taught the filter about must still be
		// hidden from a token that was not granted its scope.
		name:      "an unknown scope hides every scoped group",
		scopes:    []string{"something-else"},
		wantTools: []string{"unprefixed-read"},
	}, {
		name:      "several granted scopes show several groups",
		scopes:    []string{"pro", "projects"},
		wantTools: []string{"twpro-read", "unprefixed-read", "twprojects-read"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.scopes != nil {
				ctx = twctx.WithScopes(ctx, tt.scopes)
			}

			got := listToolNames(ctx, t, newScopedTestMCPServer(t))
			want := slices.Clone(tt.wantTools)
			slices.Sort(got)
			slices.Sort(want)

			if !slices.Equal(got, want) {
				t.Errorf("tools = %v, want %v", got, want)
			}
		})
	}
}

// newScopedTestMCPServer builds a server with three groups: one standing in for
// a premium server's own namespace, one for an existing product, and one that
// declares no namespace at all.
func newScopedTestMCPServer(t *testing.T) *mcp.Server {
	t.Helper()

	toolsets.RegisterToolOrder(nil)

	newGroup := func(toolPrefix, scope, toolName string) *toolsets.ToolsetGroup {
		toolset := toolsets.NewToolset(toolsets.Method(toolName), "toolset used by the config tests")
		toolset.AddReadTools(newTestReadTool(toolName))
		group := toolsets.NewToolsetGroup(false)
		if scope != "" {
			group.SetNamespace(toolPrefix, scope)
		}
		group.AddToolset(toolset)
		if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
			t.Fatalf("failed to enable toolsets: %v", err)
		}
		return group
	}

	var resources Resources
	resources.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewMCPServer(resources,
		newGroup("twpro", "pro", "twpro-read"),
		newGroup("twprojects", "projects", "twprojects-read"),
		// A group that declares no namespace opts out of scope filtering.
		newGroup("", "", "unprefixed-read"),
	)
}

func listToolNames(ctx context.Context, t *testing.T, server *mcp.Server) []string {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
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

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	return names
}
