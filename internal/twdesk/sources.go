package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodSourceCreate toolsets.Method = "twdesk-create_source"
	MethodSourceUpdate toolsets.Method = "twdesk-update_source"
	MethodSourceDelete toolsets.Method = "twdesk-delete_source"
	MethodSourceGet    toolsets.Method = "twdesk-get_source"
	MethodSourceList   toolsets.Method = "twdesk-list_sources"
)

func init() {
	toolsets.RegisterMethod(MethodSourceCreate)
	toolsets.RegisterMethod(MethodSourceUpdate)
	toolsets.RegisterMethod(MethodSourceDelete)
	toolsets.RegisterMethod(MethodSourceGet)
	toolsets.RegisterMethod(MethodSourceList)
}
