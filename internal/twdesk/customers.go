package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomerCreate toolsets.Method = "twdesk-create_customer"
	MethodCustomerUpdate toolsets.Method = "twdesk-update_customer"
	MethodCustomerDelete toolsets.Method = "twdesk-delete_customer"
	MethodCustomerGet    toolsets.Method = "twdesk-get_customer"
	MethodCustomerList   toolsets.Method = "twdesk-list_customers"
)
