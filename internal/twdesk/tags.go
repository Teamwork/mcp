package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTagCreate toolsets.Method = "twdesk-create_tag"
	MethodTagUpdate toolsets.Method = "twdesk-update_tag"
	MethodTagDelete toolsets.Method = "twdesk-delete_tag"
	MethodTagGet    toolsets.Method = "twdesk-get_tag"
	MethodTagList   toolsets.Method = "twdesk-list_tags"
)
