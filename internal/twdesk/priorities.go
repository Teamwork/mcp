package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodPriorityCreate toolsets.Method = "twdesk-create_priority"
	MethodPriorityUpdate toolsets.Method = "twdesk-update_priority"
	MethodPriorityDelete toolsets.Method = "twdesk-delete_priority"
	MethodPriorityGet    toolsets.Method = "twdesk-get_priority"
	MethodPriorityList   toolsets.Method = "twdesk-list_priorities"
)

func init() {
	toolsets.RegisterMethod(MethodPriorityCreate)
	toolsets.RegisterMethod(MethodPriorityUpdate)
	toolsets.RegisterMethod(MethodPriorityDelete)
	toolsets.RegisterMethod(MethodPriorityGet)
	toolsets.RegisterMethod(MethodPriorityList)
}
