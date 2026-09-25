package twdesk

import (
	"net/http"

	"github.com/teamwork/mcp/pkg/toolsets"
)

const (
	deskTicketsDescription   = "Tickets, messages, files, and inboxes in Teamwork Desk."
	deskCustomersDescription = "Companies, customers, and user management in Teamwork Desk."
	deskAdminDescription     = "Inbox configuration: priorities, statuses, types, tags, sources, custom fields " +
		"and happiness ratings in Teamwork Desk."
	deskHelpDocsDescription = "Help doc articles, categories and sites in Teamwork Desk."
)

// Sub-toolset keys for twdesk. These are the valid values for the
// -toolsets flag when selecting Teamwork Desk functionality.
const (
	// ToolsetTickets covers tickets, messages, files, and inboxes.
	ToolsetTickets toolsets.Method = "twdesk-tickets"
	// ToolsetCustomers covers companies, customers, and users.
	ToolsetCustomers toolsets.Method = "twdesk-customers"
	// ToolsetAdmin covers priorities, statuses, types, tags, sources, custom
	// fields and happiness ratings.
	ToolsetAdmin toolsets.Method = "twdesk-admin"
	// ToolsetHelpDocs covers help doc articles and sites.
	ToolsetHelpDocs toolsets.Method = "twdesk-helpdocs"
)

func init() {
	toolsets.RegisterMethod(ToolsetTickets)
	toolsets.RegisterMethod(ToolsetCustomers)
	toolsets.RegisterMethod(ToolsetAdmin)
	toolsets.RegisterMethod(ToolsetHelpDocs)
}

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Desk.
func DefaultToolsetGroup(readOnly bool, httpClient *http.Client) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly).SetNamespace("twdesk", "desk")

	// --- tickets sub-toolset ---
	group.AddToolset(toolsets.NewToolset(ToolsetTickets, deskTicketsDescription).
		AddWriteTools(
			FileCreate(httpClient),
			MessageCreate(httpClient),
			TicketCreate(httpClient),
			TicketUpdate(httpClient),
			TicketTaskLink(httpClient),
			TicketTaskUnlink(httpClient),
		).
		AddReadTools(
			InboxGet(httpClient),
			InboxList(httpClient),
			TicketGet(httpClient),
			TicketSearch(httpClient),
		))

	// --- customers sub-toolset ---
	group.AddToolset(toolsets.NewToolset(ToolsetCustomers, deskCustomersDescription).
		AddWriteTools(
			CompanyCreate(httpClient),
			CompanyUpdate(httpClient),
			CustomerCreate(httpClient),
			CustomerUpdate(httpClient),
		).
		AddReadTools(
			CompanyGet(httpClient),
			CompanyList(httpClient),
			CustomerGet(httpClient),
			CustomerList(httpClient),
			UserGet(httpClient),
			UserList(httpClient),
		))

	// --- admin sub-toolset ---
	group.AddToolset(toolsets.NewToolset(ToolsetAdmin, deskAdminDescription).
		AddWriteTools(
			PriorityCreate(httpClient),
			PriorityUpdate(httpClient),
			StatusCreate(httpClient),
			StatusUpdate(httpClient),
			TagCreate(httpClient),
			TagUpdate(httpClient),
			TypeCreate(httpClient),
			TypeUpdate(httpClient),
		).
		AddReadTools(
			CustomFieldList(httpClient),
			HappinessRatingOptionList(httpClient),
			PriorityGet(httpClient),
			PriorityList(httpClient),
			SourceList(httpClient),
			StatusGet(httpClient),
			StatusList(httpClient),
			TagGet(httpClient),
			TagList(httpClient),
			TypeGet(httpClient),
			TypeList(httpClient),
		))

	// --- helpdocs sub-toolset ---
	group.AddToolset(toolsets.NewToolset(ToolsetHelpDocs, deskHelpDocsDescription).
		AddWriteTools(
			HelpDocArticleCreate(httpClient),
			HelpDocArticleUpdate(httpClient),
		).
		AddReadTools(
			HelpDocArticleGet(httpClient),
			HelpDocArticleSearch(httpClient),
			HelpDocCategoryList(httpClient),
			HelpDocSiteGet(httpClient),
			HelpDocSiteList(httpClient),
		))

	return group
}
