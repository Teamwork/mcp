package twdesk

import (
	"net/http"

	"github.com/teamwork/mcp/internal/toolsets"
)

const (
	deskCustomersDescription = "Companies, customers, and user management in Teamwork Desk."
	deskAdminDescription     = "Inbox configuration: priorities, statuses, types, and tags in Teamwork Desk."
)

// Sub-toolset keys for twdesk. These are the valid values for the
// -toolsets flag when selecting Teamwork Desk functionality.
const (
	// ToolsetTickets covers tickets, messages, files, and inboxes.
	ToolsetTickets toolsets.Method = "twdesk-tickets"
	// ToolsetCustomers covers companies, customers, and users.
	ToolsetCustomers toolsets.Method = "twdesk-customers"
	// ToolsetAdmin covers priorities, statuses, types, and tags.
	ToolsetAdmin toolsets.Method = "twdesk-admin"
)

func init() {
	toolsets.RegisterMethod(ToolsetTickets)
	toolsets.RegisterMethod(ToolsetCustomers)
	toolsets.RegisterMethod(ToolsetAdmin)
}

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Desk.
func DefaultToolsetGroup(readOnly bool, httpClient *http.Client) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly)

	// --- tickets sub-toolset ---
	group.AddToolset(toolsets.NewToolset(ToolsetTickets, projectDescription).
		AddWriteTools(
			FileCreate(httpClient),
			MessageCreate(httpClient),
			TicketCreate(httpClient),
			TicketUpdate(httpClient),
		).
		AddReadTools(
			InboxGet(httpClient),
			InboxList(httpClient),
			TicketGet(httpClient),
			TicketList(httpClient),
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
			PriorityGet(httpClient),
			PriorityList(httpClient),
			StatusGet(httpClient),
			StatusList(httpClient),
			TagGet(httpClient),
			TagList(httpClient),
			TypeGet(httpClient),
			TypeList(httpClient),
		))

	return group
}
