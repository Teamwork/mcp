package config

import (
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/toolsets"
)

func TestOrderTools(t *testing.T) {
	toolsets.RegisterToolOrder([]toolsets.Method{
		"twprojects-get_task",
		"twprojects-list_tasks",
		"twprojects-complete_task",
		"twdesk-search_tickets",
		"twprojects-list_users",
	})

	// Preferred tools arrive out of order and mixed with unlisted ones.
	tools := []*mcp.Tool{
		{Name: "twprojects-zebra"},
		{Name: "twprojects-complete_task"},
		{Name: "twprojects-get_task"},
		{Name: "twprojects-alpha"},
		{Name: "twprojects-list_tasks"},
		{Name: "twdesk-search_tickets"},
		{Name: "twprojects-list_users"},
	}

	orderTools(tools)

	got := make([]string, len(tools))
	for i, tool := range tools {
		got[i] = tool.Name
	}

	want := []string{
		// preferred tools, in registered order
		"twprojects-get_task",
		"twprojects-list_tasks",
		"twprojects-complete_task",
		"twdesk-search_tickets",
		"twprojects-list_users",
		// remaining tools, alphabetically
		"twprojects-alpha",
		"twprojects-zebra",
	}

	if !slices.Equal(got, want) {
		t.Errorf("orderTools() = %v, want %v", got, want)
	}
}

// TestOrderToolsNoRegistration ensures tools fall back to alphabetical order
// when no preferred order has been registered.
func TestOrderToolsNoRegistration(t *testing.T) {
	toolsets.RegisterToolOrder(nil)

	tools := []*mcp.Tool{
		{Name: "twprojects-zebra"},
		{Name: "twprojects-alpha"},
		{Name: "twprojects-mango"},
	}

	orderTools(tools)

	got := make([]string, len(tools))
	for i, tool := range tools {
		got[i] = tool.Name
	}

	want := []string{"twprojects-alpha", "twprojects-mango", "twprojects-zebra"}
	if !slices.Equal(got, want) {
		t.Errorf("orderTools() = %v, want %v", got, want)
	}
}
