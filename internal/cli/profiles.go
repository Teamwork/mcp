// Package cli registers this server's toolset profiles and preferred tool
// order, then hands back the flag value from pkg/cli.
//
// The registrations are explicit rather than an init() side effect: pkg/cli is
// what parses the flag, so a server that only imported that would silently
// accept no profile name at all.
package cli

import (
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	pkgcli "github.com/teamwork/mcp/pkg/cli"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// NewMethods registers this server's profiles and tool order, then returns a
// flag value seeded with the given methods.
func NewMethods(initial ...toolsets.Method) *pkgcli.Methods {
	registerProfiles()
	registerToolOrder()
	return pkgcli.NewMethods(initial...)
}

// registerProfiles declares the named toolset collections this server exposes,
// both as -toolsets values and as URL path prefixes on the HTTP server.
func registerProfiles() {
	toolsets.RegisterProfile("project-manager", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetPlanning,
		twprojects.ToolsetContent,
	})
	toolsets.RegisterProfiles([]string{"support", "desk"}, []toolsets.Method{
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
	})
	toolsets.RegisterProfile("analyst", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetPlanning,
		twprojects.ToolsetTime,
		twprojects.ToolsetContent,
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
		twdesk.ToolsetAdmin,
	})
	toolsets.RegisterProfile("knowledge-manager", []toolsets.Method{
		twspaces.ToolsetSpaces,
		twspaces.ToolsetPages,
		twspaces.ToolsetContent,
	})
	toolsets.RegisterProfile("ops", []toolsets.Method{
		toolsets.MethodAll,
	})
}

// registerToolOrder defines the order in which tools are presented to MCP
// clients, with the most commonly used tools first. Clients that truncate the
// tool list at a fixed size keep the most useful tools; any tool not listed here
// follows alphabetically.
func registerToolOrder() {
	toolsets.RegisterToolOrder([]toolsets.Method{
		twprojects.MethodTaskGet,
		twprojects.MethodTaskList,
		twprojects.MethodTaskCreate,
		twprojects.MethodCommentList,
		twprojects.MethodTaskUpdate,
		twprojects.MethodProjectList,
		twprojects.MethodTimelogCreate,
		twprojects.MethodTimelogList,
		twprojects.MethodSearch,
		twprojects.MethodTasklistList,
		twprojects.MethodCommentCreate,
		twprojects.MethodTasklistGet,
		twprojects.MethodProjectGet,
		twdesk.MethodTicketGet,
		twprojects.MethodTimelogUpdate,
		twprojects.MethodActivityList,
		twprojects.MethodWorkflowStageTaskMove,
		twprojects.MethodTaskComplete,
		twdesk.MethodTicketSearch,
		twprojects.MethodUserList,
	})
}
