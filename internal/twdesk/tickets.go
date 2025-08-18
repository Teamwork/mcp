package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTicketCreate toolsets.Method = "twdesk-create_ticket"
	MethodTicketUpdate toolsets.Method = "twdesk-update_ticket"
	MethodTicketDelete toolsets.Method = "twdesk-delete_ticket"
	MethodTicketGet    toolsets.Method = "twdesk-get_ticket"
	MethodTicketList   toolsets.Method = "twdesk-list_tickets"
)

func init() {
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagDelete)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)
}
