package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTypeCreate toolsets.Method = "twdesk-create_type"
	MethodTypeUpdate toolsets.Method = "twdesk-update_type"
	MethodTypeDelete toolsets.Method = "twdesk-delete_type"
	MethodTypeGet    toolsets.Method = "twdesk-get_type"
	MethodTypeList   toolsets.Method = "twdesk-list_types"
)

func init() {
	toolsets.RegisterMethod(MethodTypeCreate)
	toolsets.RegisterMethod(MethodTypeUpdate)
	toolsets.RegisterMethod(MethodTypeDelete)
	toolsets.RegisterMethod(MethodTypeGet)
	toolsets.RegisterMethod(MethodTypeList)
}
