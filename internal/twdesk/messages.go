package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodMessageCreate toolsets.Method = "twdesk-create_message"
	MethodMessageUpdate toolsets.Method = "twdesk-update_message"
	MethodMessageGet    toolsets.Method = "twdesk-get_message"
	MethodMessageList   toolsets.Method = "twdesk-list_messages"
)

func init() {
	toolsets.RegisterMethod(MethodMessageCreate)
	toolsets.RegisterMethod(MethodMessageUpdate)
	toolsets.RegisterMethod(MethodMessageGet)
	toolsets.RegisterMethod(MethodMessageList)
}
