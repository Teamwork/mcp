package twdesk

import "github.com/teamwork/mcp/internal/toolsets"

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCompanyCreate toolsets.Method = "twdesk-create_company"
	MethodCompanyUpdate toolsets.Method = "twdesk-update_company"
	MethodCompanyDelete toolsets.Method = "twdesk-delete_company"
	MethodCompanyGet    toolsets.Method = "twdesk-get_company"
	MethodCompanyList   toolsets.Method = "twdesk-list_companies"
)

func init() {
	toolsets.RegisterMethod(MethodCompanyCreate)
	toolsets.RegisterMethod(MethodCompanyUpdate)
	toolsets.RegisterMethod(MethodCompanyDelete)
	toolsets.RegisterMethod(MethodCompanyGet)
	toolsets.RegisterMethod(MethodCompanyList)
}
