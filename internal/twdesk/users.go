package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodUserCreate toolsets.Method = "twdesk-create_user"
	MethodUserUpdate toolsets.Method = "twdesk-update_user"
	MethodUserDelete toolsets.Method = "twdesk-delete_user"
	MethodUserGet    toolsets.Method = "twdesk-get_user"
	MethodUserList   toolsets.Method = "twdesk-list_users"
)

func init() {
	toolsets.RegisterMethod(MethodUserCreate)
	toolsets.RegisterMethod(MethodUserUpdate)
	toolsets.RegisterMethod(MethodUserDelete)
	toolsets.RegisterMethod(MethodUserGet)
	toolsets.RegisterMethod(MethodUserList)
}
